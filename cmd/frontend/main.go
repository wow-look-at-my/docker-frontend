package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/frontend/gateway/grpcclient"
	"github.com/moby/buildkit/util/appcontext"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/wow-look-at-my/docker-frontend/pkg/converter"
	"github.com/wow-look-at-my/docker-frontend/pkg/parser"
)

func main() {
	if err := grpcclient.RunFromEnvironment(appcontext.Context(), build); err != nil {
		log.Fatalf("frontend error: %v", err)
	}
}

func build(ctx context.Context, c client.Client) (*client.Result, error) {
	opts := c.BuildOpts().Opts
	filename := opts["filename"]
	if filename == "" {
		filename = "Dockerfile"
	}

	// Build context (user's files)
	buildContext := llb.Local("context",
		llb.SharedKeyHint("dockerfile-frontend"),
	)

	// Read the Dockerfile content
	dockerfileSrc := llb.Local("dockerfile",
		llb.IncludePatterns([]string{filename}),
		llb.SharedKeyHint("dockerfile"),
	)

	def, err := dockerfileSrc.Marshal(ctx)
	if err != nil {
		return nil, err
	}

	dtRes, err := c.Solve(ctx, client.SolveRequest{
		Definition: def.ToPB(),
	})
	if err != nil {
		return nil, err
	}

	ref, err := dtRes.SingleRef()
	if err != nil {
		return nil, err
	}

	dtBytes, err := ref.ReadFile(ctx, client.ReadRequest{
		Filename: filename,
	})
	if err != nil {
		return nil, err
	}

	// Parse the DSL
	insts, err := parser.Parse(string(dtBytes))
	if err != nil {
		return nil, err
	}

	// Convert to LLB, passing the build context for COPY instructions
	convResult, err := converter.Convert(insts, buildContext)
	if err != nil {
		return nil, err
	}

	// Marshal and solve the final state
	finalDef, err := convResult.State.Marshal(ctx)
	if err != nil {
		return nil, err
	}

	res, err := c.Solve(ctx, client.SolveRequest{
		Definition: finalDef.ToPB(),
	})
	if err != nil {
		return nil, err
	}

	// Set image config on the result
	imgConfig := v1.Image{
		Config: convResult.Image.Config,
	}
	configBytes, err := json.Marshal(imgConfig)
	if err != nil {
		return nil, err
	}

	res.AddMeta("containerimage.config", configBytes)

	return res, nil
}
