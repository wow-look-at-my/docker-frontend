package main

import (
	"context"
	"log"
	"maps"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/frontend/gateway/grpcclient"
	gwpb "github.com/moby/buildkit/frontend/gateway/pb"
	"github.com/moby/buildkit/util/appcontext"
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

	// Create an LLB state containing the preprocessed Dockerfile
	dockerfileSt := llb.Scratch().File(llb.Mkfile("Dockerfile", 0644, []byte(preprocessed)))

	// Delegate to the standard dockerfile builder, providing preprocessed
	// content via a client wrapper that injects it as a frontend input
	return builder.Build(ctx, &clientWithPreprocessed{
		Client:       c,
		dockerfileSt: dockerfileSt,
	})
}

// clientWithPreprocessed wraps a gateway client to inject preprocessed
// Dockerfile content via the Inputs method, avoiding FrontendInputs which
// is not supported by all BuildKit daemon versions.
type clientWithPreprocessed struct {
	client.Client
	dockerfileSt llb.State
}

func (c *clientWithPreprocessed) BuildOpts() client.BuildOpts {
	opts := c.Client.BuildOpts()
	// Advertise frontend inputs capability so the dockerfile builder
	// reads from our injected inputs instead of the local source
	opts.Caps = gwpb.Caps.CapSet(gwpb.Caps.All())
	// Override filename since our preprocessed input uses "Dockerfile"
	opts.Opts = maps.Clone(opts.Opts)
	opts.Opts["filename"] = "Dockerfile"
	return opts
}

func (c *clientWithPreprocessed) Inputs(ctx context.Context) (map[string]llb.State, error) {
	return map[string]llb.State{
		"dockerfile": c.dockerfileSt,
	}, nil
}
