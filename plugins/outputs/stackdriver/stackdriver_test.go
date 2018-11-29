package stackdriver

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"testing"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"github.com/golang/protobuf/proto"
	emptypb "github.com/golang/protobuf/ptypes/empty"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/testutil"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// clientOpt is the option tests should use to connect to the test server.
// It is initialized by TestMain.
var clientOpt option.ClientOption

var mockMetric mockMetricServer

type mockMetricServer struct {
	// Embed for forward compatibility.
	// Tests will keep working if more methods are added
	// in the future.
	monitoringpb.MetricServiceServer

	reqs []proto.Message

	// If set, all calls return this error.
	err error

	// responses to return if err == nil
	resps []proto.Message
}

func (s *mockMetricServer) CreateTimeSeries(ctx context.Context, req *monitoringpb.CreateTimeSeriesRequest) (*emptypb.Empty, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*emptypb.Empty), nil
}

func TestMain(m *testing.M) {
	serv := grpc.NewServer()
	monitoringpb.RegisterMetricServiceServer(serv, &mockMetric)

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatal(err)
	}
	go serv.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	clientOpt = option.WithGRPCConn(conn)

	os.Exit(m.Run())
}

func TestWrite(t *testing.T) {
	expectedResponse := &emptypb.Empty{}
	mockMetric.err = nil
	mockMetric.reqs = nil
	mockMetric.resps = append(mockMetric.resps[:0], expectedResponse)

	c, err := monitoring.NewMetricClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	s := &Stackdriver{
		Project:   fmt.Sprintf("projects/%s", "[PROJECT]"),
		Namespace: "test",
		client:    c,
	}

	err = s.Connect()
	require.NoError(t, err)
	err = s.Write(testutil.MockMetrics())
	require.NoError(t, err)
}

func TestGetStackdriverLabels(t *testing.T) {
	tags := []*telegraf.Tag{
		{Key: "project", Value: "bar"},
		{Key: "discuss", Value: "revolutionary"},
		{Key: "marble", Value: "discount"},
		{Key: "applied", Value: "falsify"},
		{Key: "test", Value: "foo"},
		{Key: "porter", Value: "discount"},
		{Key: "play", Value: "tiger"},
		{Key: "fireplace", Value: "display"},
		{Key: "host", Value: "this"},
		{Key: "name", Value: "bat"},
		{Key: "device", Value: "local"},
		{Key: "reserve", Value: "publication"},
		{Key: "xpfqacltlmpguimhtjlou2qlmf9uqqwk3teajwlwqkoxtsppbnjksaxvzc1aa973pho9m96gfnl5op8ku7sv93rexyx42qe3zty12ityv", Value: "keyquota"},
		{Key: "valuequota", Value: "icym5wcpejnhljcvy2vwk15svmhrtueoppwlvix61vlbaeedufn1g6u4jgwjoekwew9s2dboxtgrkiyuircnl8h1lbzntt9gzcf60qunhxurhiz0g2bynzy1v6eyn4ravndeiiugobsrsj2bfaguahg4gxn7nx4irwfknunhkk6jdlldevawj8levebjajcrcbeugewd14fa8o34ycfwx2ymalyeqxhfqrsksxnii2deqq6cghrzi6qzwmittkzdtye3imoygqmjjshiskvnzz1e4ipd9c6wfor5jsygn1kvcg6jm4clnsl1fnxotbei9xp4swrkjpgursmfmkyvxcgq9hoy435nwnolo3ipnvdlhk6pmlzpdjn6gqi3v9gv7jn5ro2p1t5ufxzfsvqq1fyrgoi7gvmttil1banh3cftkph1dcoaqfhl7y0wkvhwwvrmslmmxp1wedyn8bacd7akmjgfwdvcmrymbzvmrzfvq1gs1xnmmg8rsfxci2h6r1ralo3splf4f3bdg4c7cy0yy9qbxzxhcmdpwekwc7tdjs8uj6wmofm2aor4hum8nwyfwwlxy3yvsnbjy32oucsrmhcnu6l2i8laujkrhvsr9fcix5jflygznlydbqw5uhw1rg1g5wiihqumwmqgggemzoaivm3ut41vjaff4uqtqyuhuwblmuiphfkd7si49vgeeswzg7tpuw0oxmkesgibkcjtev2h9ouxzjs3eb71jffhdacyiuyhuxwvm5bnrjewbm4x2kmhgbirz3eoj7ijgplggdkx5vixufg65ont8zi1jabsuxx0vsqgprunwkugqkxg2r7iy6fmgs4lob4dlseinowkst6gp6x1ejreauyzjz7atzm3hbmr5rbynuqp4lxrnhhcbuoun69mavvaaki0bdz5ybmbbbz5qdv0odtpjo2aezat5uosjuhzbvic05jlyclikynjgfhencdkz3qcqzbzhnsynj1zdke0sk4zfpvfyryzsxv9pu0qm"},
	}

	labels := getStackdriverLabels(tags)
	require.Equal(t, QuotaLabelsPerMetricDescriptor, len(labels))
}