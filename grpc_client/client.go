package grpc_client

import (
	"context"
	"io"
	"time"

	"github.com/f-taxes/german_tax_report/global"
	"github.com/f-taxes/german_tax_report/proto"
	"github.com/kataras/golog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

var GrpcClient *FTaxesClient

type FTaxesClient struct {
	conStr     string
	Connection *grpc.ClientConn
	GrpcClient proto.FTaxesClient
}

func NewFTaxesClient(conStr string) *FTaxesClient {
	return &FTaxesClient{
		conStr: conStr,
	}
}

func (c *FTaxesClient) Connect(ctx context.Context) error {
	con, err := grpc.DialContext(ctx, c.conStr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithConnectParams(grpc.ConnectParams{
		MinConnectTimeout: time.Second * 30,
		Backoff:           backoff.Config{MaxDelay: time.Second},
	}))

	if err != nil {
		golog.Errorf("Failed to establish grpc connections: %v", err)
		return err
	}

	go func() {
		state := con.GetState()
		for {
			golog.Infof("Connection state: %s", state.String())
			con.WaitForStateChange(context.Background(), state)
			state = con.GetState()
		}
	}()

	c.Connection = con
	c.GrpcClient = proto.NewFTaxesClient(con)

	return nil
}

func (c *FTaxesClient) ShowJobProgress(ctx context.Context, job *proto.JobProgress) error {
	job.Plugin = global.Plugin.Label
	_, err := c.GrpcClient.ShowJobProgress(ctx, job)
	return err
}

func (c *FTaxesClient) StreamRecords(ctx context.Context, job *proto.StreamRecordsJob, out chan *proto.Record) error {
	stream, err := c.GrpcClient.StreamRecords(ctx, job)

	if err != nil {
		return err
	}

	done := make(chan bool)

	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				done <- true
				return
			}

			if err != nil {
				golog.Errorf("Error while streaming records: %v", err)
				return
			}

			out <- resp
		}
	}()

	<-done
	return nil
}
