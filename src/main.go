package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	pb "emqx.io/grpc/exhook/protobuf"
	"google.golang.org/grpc"
)

// server is used to implement emqx_exhook_v1.s *server
type server struct {
	pb.UnimplementedHookProviderServer
}

// HookProviderServer callbacks

func (s *server) OnProviderLoaded(ctx context.Context, in *pb.ProviderLoadedRequest) (*pb.LoadedResponse, error) {

	fmt.Println("OnProviderLoaded called..")

	hooks := []*pb.HookSpec{
		{Name: "message.publish"},
	}

	return &pb.LoadedResponse{Hooks: hooks}, nil
}

func MillisToTime(ms int64) time.Time {
	const msInSecond int64 = 1e3
	const nsInMillisecond int64 = 1e6

	return time.Unix(ms/msInSecond, (ms%msInSecond)*nsInMillisecond)
}

func (s *server) OnMessagePublish(ctx context.Context, in *pb.MessagePublishRequest) (*pb.ValuedResponse, error) {

	var msg interface{}
	err := json.Unmarshal(in.Message.Payload, &msg)

	// no valid JSON? --> return original message!
	if err != nil {
		reply := &pb.ValuedResponse{}
		reply.Type = pb.ValuedResponse_STOP_AND_RETURN
		reply.Value = &pb.ValuedResponse_Message{Message: in.Message}
		return reply, nil
	}

	jsonMap := msg.(map[string]interface{})

	// timestamp already present? --> return original message!
	_, ok := jsonMap["__t"]
	if ok {
		reply := &pb.ValuedResponse{}
		reply.Type = pb.ValuedResponse_STOP_AND_RETURN
		reply.Value = &pb.ValuedResponse_Message{Message: in.Message}
		return reply, nil
	}

	timestamp := MillisToTime(int64(in.Message.Timestamp))
	jsonMap["__t"] = timestamp.UTC().Format("2006-01-02T15:04:05.000Z07:00")
	// jsonMap["__t"] = timestamp.UTC().Format(time.RFC3339) // -> ".millis" missing

	payload, err := json.Marshal(jsonMap)
	// everything fine?
	if err == nil {
		// .. overwrite the payload though!
		in.Message.Payload = payload
	}

	reply := &pb.ValuedResponse{}
	reply.Type = pb.ValuedResponse_STOP_AND_RETURN
	reply.Value = &pb.ValuedResponse_Message{Message: in.Message}
	return reply, nil
}

func main() {

	port := os.Getenv("GRPC_PORT")
	_, err := strconv.ParseUint(port, 10, 16)

	// valid port via ENV set?
	if port == "" || err != nil {
		log.Printf(
			"invalid GRPC_PORT '%s', defaulting to 9531 instead.\n",
			port,
		)

		//.. nope? use default one!
		port = "9531"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v\n", err)
	}

	s := grpc.NewServer()
	pb.RegisterHookProviderServer(s, &server{})

	log.Printf("Starting gRPC server on ::%s.\n", port)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v\n", err)
	}
}
