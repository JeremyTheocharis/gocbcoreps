package gocbps

import (
	"crypto/x509"
	"fmt"
	"strings"

	"github.com/couchbase/goprotostellar/genproto/analytics_v1"
	"github.com/couchbase/goprotostellar/genproto/kv_v1"
	"github.com/couchbase/goprotostellar/genproto/query_v1"
	"github.com/couchbase/goprotostellar/genproto/routing_v1"
	"github.com/couchbase/goprotostellar/genproto/search_v1"
	"github.com/couchbase/goprotostellar/genproto/view_v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn            *grpc.ClientConn
	routingClient   routing_v1.RoutingServiceClient
	kvClient        kv_v1.KvServiceClient
	queryClient     query_v1.QueryServiceClient
	searchClient    search_v1.SearchServiceClient
	analyticsClient analytics_v1.AnalyticsServiceClient
	viewClient      view_v1.ViewServiceClient
}

type ConnectOptions struct {
	Username          string
	Password          string
	ClientCertificate *x509.CertPool
	ServerURL         string
}

func Connect(connStr string, opts *ConnectOptions) (*Client, error) {
	var transportDialOpt grpc.DialOption
	var perRpcDialOpt grpc.DialOption

	// All endpoints are auth protected!
	if opts.Username == "" || opts.Password == "" {
		return nil, fmt.Errorf("username and password must not be empty")
	}

	if opts.ClientCertificate != nil && (opts.Username != "" && opts.Password != "") {
		transportDialOpt = grpc.WithTransportCredentials(
			credentials.NewClientTLSFromCert(opts.ClientCertificate, opts.ServerURL))
		basicAuthCreds, err := newGrpcBasicAuth(opts.Username, opts.Password)
		if err != nil {
			return nil, err
		}
		perRpcDialOpt = grpc.WithPerRPCCredentials(basicAuthCreds)
	} else {
		// When gateway is not TLS enabled
		transportDialOpt = grpc.WithTransportCredentials(insecure.NewCredentials())
		basicAuthCreds, err := newGrpcBasicAuth(opts.Username, opts.Password)
		if err != nil {
			return nil, err
		}
		perRpcDialOpt = grpc.WithPerRPCCredentials(basicAuthCreds)
	}

	// use port 18091 by default
	connStrPieces := strings.Split(connStr, ":")
	if len(connStrPieces) == 1 {
		connStrPieces = append(connStrPieces, "18098")
	}
	connStr = strings.Join(connStrPieces, ":")

	dialOpts := []grpc.DialOption{transportDialOpt}
	if perRpcDialOpt != nil {
		dialOpts = append(dialOpts, perRpcDialOpt)
	}

	conn, err := grpc.Dial(connStr, dialOpts...)
	if err != nil {
		return nil, err
	}

	routingClient := routing_v1.NewRoutingServiceClient(conn)
	kvClient := kv_v1.NewKvServiceClient(conn)
	queryClient := query_v1.NewQueryServiceClient(conn)
	searchClient := search_v1.NewSearchServiceClient(conn)
	analyticsClient := analytics_v1.NewAnalyticsServiceClient(conn)
	viewClient := view_v1.NewViewServiceClient(conn)

	return &Client{
		conn:            conn,
		routingClient:   routingClient,
		kvClient:        kvClient,
		queryClient:     queryClient,
		searchClient:    searchClient,
		analyticsClient: analyticsClient,
		viewClient:      viewClient,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Bucket(bucketName string) *Bucket {
	return &Bucket{
		client:     c,
		bucketName: bucketName,
	}
}

// INTERNAL: Used for testing
func (c *Client) GetConn() *grpc.ClientConn {
	return c.conn
}
