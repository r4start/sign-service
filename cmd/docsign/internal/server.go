package internal

import (
	"context"
	"errors"
	"io"

	ed255192 "golang.org/x/crypto/ed25519"

	pb "github.com/r4start/sign-service/pkg/proto"
)

type GrpcDocSignServer struct {
	pb.UnimplementedSignServiceServer

	privateKey ed255192.PrivateKey
	publicKey  ed255192.PublicKey
}

func NewSignServer(privateKey ed255192.PrivateKey, publicKey ed255192.PublicKey) (*GrpcDocSignServer, error) {
	return &GrpcDocSignServer{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

func (server *GrpcDocSignServer) Sign(_ context.Context, doc *pb.Document) (*pb.DocSign, error) {
	return &pb.DocSign{Sign: ed255192.Sign(server.privateKey, doc.Data)}, nil
}

func (server *GrpcDocSignServer) Verify(_ context.Context, req *pb.VerifyRequest) (*pb.VerifyResponse, error) {
	return &pb.VerifyResponse{IsOk: ed255192.Verify(server.publicKey, req.Doc.Data, req.Sign.Sign)}, nil
}

func (server *GrpcDocSignServer) SignBatch(_ context.Context, docs *pb.DocumentBatch) (*pb.DocSignBatch, error) {
	signs := &pb.DocSignBatch{Sign: make([][]byte, len(docs.Doc))}
	for i, doc := range docs.Doc {
		signs.Sign[i] = ed255192.Sign(server.privateKey, doc)
	}
	return signs, nil
}
func (server *GrpcDocSignServer) VerifyBatch(_ context.Context, signs *pb.VerifyBatchRequest) (*pb.VerifyBatchResponse, error) {
	response := &pb.VerifyBatchResponse{Status: make([]bool, len(signs.Docs))}
	for i, sign := range signs.Docs {
		response.Status[i] = ed255192.Verify(server.publicKey, sign.Doc.Data, sign.Sign.Sign)
	}
	return response, nil
}

func (server *GrpcDocSignServer) SignStream(stream pb.SignService_SignStreamServer) error {
	for {
		doc, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		}

		sign := ed255192.Sign(server.privateKey, doc.Data)
		if err := stream.Send(&pb.DocSign{Sign: sign}); err != nil {
			return err
		}
	}
}

func (server *GrpcDocSignServer) VerifyStream(stream pb.SignService_VerifyStreamServer) error {
	for {
		doc, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		}

		result := ed255192.Verify(server.publicKey, doc.Doc.Data, doc.Sign.Sign)
		if err := stream.Send(&pb.VerifyResponse{IsOk: result}); err != nil {
			return err
		}
	}
}
