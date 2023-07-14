package internal

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"testing"

	"golang.org/x/crypto/ed25519"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/stretchr/testify/assert"

	pb "github.com/r4start/sign-service/pkg/proto"
)

func serve(t *testing.T, ctx context.Context) (pb.SignServiceClient, func()) {
	const bufSize = 1024 * 1024

	lis := bufconn.Listen(bufSize)

	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)

	service, err := NewSignServer(privateKey, publicKey)
	assert.NoError(t, err)

	server := grpc.NewServer()
	pb.RegisterSignServiceServer(server, service)

	go func(t *testing.T) {
		err := server.Serve(lis)
		assert.NoError(t, err)
	}(t)

	conn, err := grpc.DialContext(ctx, "", grpc.WithContextDialer(
		func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)

	closer := func() {
		err := lis.Close()
		assert.NoError(t, err)
		server.Stop()
	}

	client := pb.NewSignServiceClient(conn)

	return client, closer
}

func randData(t *testing.T, size int) []byte {
	b := make([]byte, size)
	_, err := rand.Read(b)
	assert.NoError(t, err)
	return b
}

func TestGrpcDocSignServer_Sign(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client, closer := serve(t, ctx)
	defer closer()

	tests := map[string]*pb.Document{
		"Case #1": {Data: randData(t, 17)},
		"Case #2": {Data: randData(t, 1024)},
		"Case #3": {Data: randData(t, 1024*1024)},
		"Case #4": {Data: nil},
	}

	for scenario, tt := range tests {
		t.Run(scenario, func(t *testing.T) {
			sign, err := client.Sign(ctx, tt)
			assert.NoError(t, err)

			verification, err := client.Verify(ctx, &pb.VerifyRequest{Sign: sign, Doc: tt})
			assert.NoError(t, err)
			assert.True(t, verification.IsOk)
		})
	}
}

func TestGrpcDocSignServer_SignBatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		shouldFail bool
		docs       *pb.DocumentBatch
	}{
		{
			name:       "Case #1",
			shouldFail: false,
			docs:       &pb.DocumentBatch{Doc: nil},
		},
		{
			name:       "Case #2",
			shouldFail: false,
			docs:       &pb.DocumentBatch{Doc: [][]byte{randData(t, 17)}},
		},
		{
			name:       "Case #3",
			shouldFail: false,
			docs:       &pb.DocumentBatch{Doc: [][]byte{randData(t, 17), randData(t, 17), randData(t, 17), randData(t, 17), randData(t, 17), randData(t, 17)}},
		},
		{
			name:       "Case #4",
			shouldFail: false,
			docs:       &pb.DocumentBatch{Doc: [][]byte{randData(t, 1024), randData(t, 1024), randData(t, 1024), randData(t, 1024), randData(t, 1024), randData(t, 17)}},
		},
		{
			name:       "Case #5",
			shouldFail: true,
			docs:       &pb.DocumentBatch{Doc: [][]byte{randData(t, 1024*1024), randData(t, 1024*1024), randData(t, 1024*1024), randData(t, 1024*1024), randData(t, 1024*1024), randData(t, 1024*1024), randData(t, 1024*1024), randData(t, 1024*1024), randData(t, 1024*1024), randData(t, 1024*1024), randData(t, 1024*1024), randData(t, 1024*1024), randData(t, 1024)}},
		},
	}

	ctx := context.Background()
	client, closer := serve(t, ctx)
	defer closer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signs, err := client.SignBatch(ctx, tt.docs)
			if tt.shouldFail {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			request := &pb.VerifyBatchRequest{
				Docs: make([]*pb.VerifyRequest, len(signs.Sign)),
			}

			for i, doc := range tt.docs.Doc {
				request.Docs[i] = &pb.VerifyRequest{Doc: &pb.Document{Data: doc}, Sign: &pb.DocSign{Sign: signs.Sign[i]}}
			}

			verification, err := client.VerifyBatch(ctx, request)

			assert.NoError(t, err)
			for _, result := range verification.Status {
				assert.True(t, result)
			}

		})
	}
}

func TestGrpcDocSignServer_SignStream(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		docs []*pb.Document
	}{
		{
			name: "Case #1",
			docs: []*pb.Document{},
		},
		{
			name: "Case #2",
			docs: []*pb.Document{
				{Data: randData(t, 17)},
			},
		},
		{
			name: "Case #3",
			docs: []*pb.Document{
				{Data: randData(t, 17)},
				{Data: randData(t, 17)},
				{Data: randData(t, 17)},
				{Data: randData(t, 17)},
			},
		},
		{
			name: "Case #4",
			docs: []*pb.Document{
				{Data: randData(t, 1024)},
				{Data: randData(t, 1024)},
				{Data: randData(t, 1024)},
				{Data: randData(t, 17)},
			},
		},
		{
			name: "Case #5",
			docs: []*pb.Document{
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
			},
		},
	}

	ctx := context.Background()
	client, closer := serve(t, ctx)
	defer closer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream, err := client.SignStream(ctx)
			assert.NoError(t, err)

			signs := make([]*pb.DocSign, 0, len(tt.docs))

			for _, doc := range tt.docs {
				assert.NoError(t, stream.Send(doc))
				sign, err := stream.Recv()
				assert.NoError(t, err)
				signs = append(signs, sign)
			}

			verifyStream, err := client.VerifyStream(ctx)
			assert.NoError(t, err)

			for index, doc := range tt.docs {
				assert.NoError(t, verifyStream.Send(&pb.VerifyRequest{Doc: doc, Sign: signs[index]}))
				check, err := verifyStream.Recv()
				assert.NoError(t, err)
				assert.True(t, check.IsOk)
			}
		})
	}
}

func TestGrpcDocSignServer_SignStream_OrderingCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		docs []*pb.Document
	}{
		{
			name: "Case #1",
			docs: []*pb.Document{},
		},
		{
			name: "Case #2",
			docs: []*pb.Document{
				{Data: randData(t, 17)},
			},
		},
		{
			name: "Case #3",
			docs: []*pb.Document{
				{Data: randData(t, 17)},
				{Data: randData(t, 17)},
				{Data: randData(t, 17)},
				{Data: randData(t, 17)},
			},
		},
		{
			name: "Case #4",
			docs: []*pb.Document{
				{Data: randData(t, 1024)},
				{Data: randData(t, 1024)},
				{Data: randData(t, 1024)},
				{Data: randData(t, 17)},
			},
		},
		{
			name: "Case #5",
			docs: []*pb.Document{
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
				{Data: randData(t, 1024*1024)},
			},
		},
	}

	ctx := context.Background()
	client, closer := serve(t, ctx)
	defer closer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream, err := client.SignStream(ctx)
			assert.NoError(t, err)

			signs := make([]*pb.DocSign, 0, len(tt.docs))

			for _, doc := range tt.docs {
				assert.NoError(t, stream.Send(doc))
			}

			for i := 0; i < len(tt.docs); i++ {
				sign, err := stream.Recv()
				assert.NoError(t, err)
				signs = append(signs, sign)
			}

			verifyStream, err := client.VerifyStream(ctx)
			assert.NoError(t, err)

			for index, doc := range tt.docs {
				assert.NoError(t, verifyStream.Send(&pb.VerifyRequest{Doc: doc, Sign: signs[index]}))
				check, err := verifyStream.Recv()
				assert.NoError(t, err)
				assert.True(t, check.IsOk)
			}
		})
	}
}

func BenchmarkGrpcDocSignServer_Sign(b *testing.B) {
	b.StopTimer()

	ctx := context.Background()
	client, closer := serve(nil, ctx)
	defer closer()

	tests := map[string]*pb.Document{
		"17":        {Data: randData(nil, 17)},
		"1024":      {Data: randData(nil, 1024)},
		"1024*1024": {Data: randData(nil, 1024*1024)},
	}

	b.StartTimer()
	for name, t := range tests {
		b.Run(fmt.Sprintf("size_%s", name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := client.Sign(ctx, t); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGrpcDocSignServer_SignUnary(b *testing.B) {
	b.StopTimer()

	ctx := context.Background()
	client, closer := serve(nil, ctx)
	defer closer()

	tests := map[string]*pb.Document{
		"17":        {Data: randData(nil, 17)},
		"1024":      {Data: randData(nil, 1024)},
		"1024*1024": {Data: randData(nil, 1024*1024)},
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for _, t := range tests {
			if _, err := client.Sign(ctx, t); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkGrpcDocSignServer_SignBatch(b *testing.B) {
	b.StopTimer()

	ctx := context.Background()
	client, closer := serve(nil, ctx)
	defer closer()

	tests := []struct {
		name string
		docs *pb.DocumentBatch
	}{
		{
			name: "0",
			docs: &pb.DocumentBatch{Doc: nil},
		},
		{
			name: "17",
			docs: &pb.DocumentBatch{Doc: [][]byte{randData(nil, 17)}},
		},
		{
			name: "6*17",
			docs: &pb.DocumentBatch{Doc: [][]byte{randData(nil, 17), randData(nil, 17), randData(nil, 17), randData(nil, 17), randData(nil, 17), randData(nil, 17)}},
		},
		{
			name: "5*1024+17",
			docs: &pb.DocumentBatch{Doc: [][]byte{randData(nil, 1024), randData(nil, 1024), randData(nil, 1024), randData(nil, 1024), randData(nil, 1024), randData(nil, 17)}},
		},
		{
			name: "17+1024+1024*1024",
			docs: &pb.DocumentBatch{Doc: [][]byte{randData(nil, 17), randData(nil, 1024), randData(nil, 1024*1024)}},
		},
	}

	b.StartTimer()
	for _, tt := range tests {
		b.Run(fmt.Sprintf("size_%s", tt.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := client.SignBatch(ctx, tt.docs); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGrpcDocSignServer_SignStream(b *testing.B) {
	b.StopTimer()

	ctx := context.Background()
	client, closer := serve(nil, ctx)
	defer closer()

	tests := []struct {
		name string
		docs []*pb.Document
	}{
		{
			name: "none",
			docs: []*pb.Document{},
		},
		{
			name: "17",
			docs: []*pb.Document{
				{Data: randData(nil, 17)},
			},
		},
		{
			name: "4*17",
			docs: []*pb.Document{
				{Data: randData(nil, 17)},
				{Data: randData(nil, 17)},
				{Data: randData(nil, 17)},
				{Data: randData(nil, 17)},
			},
		},
		{
			name: "3*1024+17",
			docs: []*pb.Document{
				{Data: randData(nil, 1024)},
				{Data: randData(nil, 1024)},
				{Data: randData(nil, 1024)},
				{Data: randData(nil, 17)},
			},
		},
		{
			name: "20*1024*1024",
			docs: []*pb.Document{
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
				{Data: randData(nil, 1024*1024)},
			},
		},
		{
			name: "1024*1024+1024+17",
			docs: []*pb.Document{
				{Data: randData(nil, 17)},
				{Data: randData(nil, 1024)},
				{Data: randData(nil, 1024*1024)},
			},
		},
	}

	b.StartTimer()
	for _, tt := range tests {
		b.Run(fmt.Sprintf("size_%s", tt.name), func(b *testing.B) {
			stream, err := client.SignStream(ctx)
			if err != nil {
				b.Fatal(err)
			}
			for i := 0; i < b.N; i++ {
				for _, doc := range tt.docs {
					if err := stream.Send(doc); err != nil {
						b.Fatal(err)
					}
				}

				for i := 0; i < len(tt.docs); i++ {
					if _, err := stream.Recv(); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}
