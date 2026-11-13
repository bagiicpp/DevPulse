package main

import (
	"context"
	"log"
	"net"
	"os"
	"strings"
	"time"

	aipb "devpulse/backend/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const aiTimeout = 35 * time.Second

type aiServer struct {
	aipb.UnimplementedAIServiceServer
}

func (s *aiServer) Analyze(ctx context.Context, request *aipb.AnalyzeRequest) (*aipb.AnalyzeResponse, error) {
	repository := request.GetContext()
	question := strings.TrimSpace(request.GetQuestion())
	if repository == nil || !repositoryName.MatchString(repository.Repository) || question == "" || len(question) > 16<<10 {
		return nil, status.Error(codes.InvalidArgument, "Use owner/repository and provide a question of at most 16 KiB")
	}
	pathBytes := 0
	for _, path := range repository.Files {
		pathBytes += len(path)
	}
	if len(repository.Files) > 1000 || pathBytes > 64<<10 || len(repository.Readme) > 64<<10 || len(repository.Ref) > 1024 {
		return nil, status.Error(codes.InvalidArgument, "Repository context exceeds the supported limits")
	}
	ctx, cancel := context.WithTimeout(ctx, aiTimeout)
	defer cancel()
	response, err := askAI(ctx, RepositoryContext{
		Repository: repository.Repository, Ref: repository.Ref, Files: append([]string{}, repository.Files...),
		FilesTruncated: repository.FilesTruncated, Readme: repository.Readme, ReadmeTruncated: repository.ReadmeTruncated,
	}, question)
	if err != nil {
		if ctx.Err() != nil {
			return nil, status.FromContextError(ctx.Err()).Err()
		}
		log.Printf("AI provider error: %v", err)
		return nil, status.Error(codes.Internal, "AI provider could not generate a valid response")
	}
	return &aipb.AnalyzeResponse{Summary: response.Summary, Findings: response.Findings, Recommendations: response.Recommendations}, nil
}

func newAIClient() (*grpc.ClientConn, error) {
	address := strings.TrimSpace(os.Getenv("AI_GRPC_ADDRESS"))
	if address == "" {
		address = "127.0.0.1:50051"
	}
	// shortcut: plaintext gRPC is for loopback/private Docker networking, add TLS and authentication before remote hosting.
	return grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1<<20)))
}

func (a *authServer) analyze(ctx context.Context, repository RepositoryContext, question string) (ChatResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, aiTimeout)
	defer cancel()
	response, err := a.ai.Analyze(ctx, &aipb.AnalyzeRequest{
		Context: &aipb.RepositoryContext{
			Repository: repository.Repository, Ref: repository.Ref, Files: repository.Files,
			FilesTruncated: repository.FilesTruncated, Readme: repository.Readme, ReadmeTruncated: repository.ReadmeTruncated,
		},
		Question: question,
	})
	if err != nil {
		return ChatResponse{}, err
	}
	// Protobuf omits empty repeated fields; the HTTP API promises JSON arrays, never null.
	return ChatResponse{
		Summary: response.Summary, Findings: append([]string{}, response.Findings...),
		Recommendations: append([]string{}, response.Recommendations...),
	}, nil
}

func serveAI() error {
	address := strings.TrimSpace(os.Getenv("AI_GRPC_LISTEN"))
	if address == "" {
		address = "127.0.0.1:50051"
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()
	log.Printf("DevPulse AI gRPC service listening on %s", address)
	return newAIServer().Serve(listener)
}

func newAIServer() *grpc.Server {
	server := grpc.NewServer(grpc.MaxRecvMsgSize(256<<10), grpc.MaxSendMsgSize(1<<20))
	aipb.RegisterAIServiceServer(server, &aiServer{})
	return server
}
