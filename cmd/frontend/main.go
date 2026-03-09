package main

import (
	"context"
	"log"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/frontend/gateway/grpcclient"
	"github.com/moby/buildkit/util/appcontext"
	pb "github.com/moby/buildkit/solver/pb"
	"github.com/wow-look-at-my/docker-frontend/pkg/converter"
)

func main() {
	if err := grpcclient.RunFromEnvironment(appcontext.Context(), build); err != nil {
		log.Fatalf("frontend error: %v", err)
	}
}

func build(ctx context.Context, c client.Client) (*client.Result, error) {
	filename := c.BuildOpts().Opts["filename"]
	if filename == "" {
		filename = "Dockerfile"
	}

	def, err := llb.Local("dockerfile", llb.IncludePatterns([]string{filename}), llb.SharedKeyHint("dockerfile")).Marshal(ctx)
	if err != nil {
		return nil, err
	}

	dtRes, err := c.Solve(ctx, client.SolveRequest{Definition: def.ToPB()})
	if err != nil {
		return nil, err
	}

	ref, err := dtRes.SingleRef()
	if err != nil {
		return nil, err
	}

	dtBytes, err := ref.ReadFile(ctx, client.ReadRequest{Filename: filename})
	if err != nil {
		return nil, err
	}

	preprocessed, err := converter.Preprocess(string(dtBytes))
	if err != nil {
		return nil, err
	}

	preprocessedDef, err := llb.Scratch().File(llb.Mkfile(filename, 0644, []byte(preprocessed))).Marshal(ctx)
	if err != nil {
		return nil, err
	}

	buildContextDef, err := llb.Local("context", llb.SharedKeyHint("dockerfile-frontend")).Marshal(ctx)
	if err != nil {
		return nil, err
	}

	return c.Solve(ctx, client.SolveRequest{
		Frontend: "dockerfile.v0",
		FrontendInputs: map[string]*pb.Definition{
			"dockerfile": preprocessedDef.ToPB(),
			"context":    buildContextDef.ToPB(),
		},
	})
}
