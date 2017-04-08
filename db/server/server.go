package api

import (
	"net"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"gitlab.com/SporeDB/sporedb/db"
	"gitlab.com/SporeDB/sporedb/db/api"
	"gitlab.com/SporeDB/sporedb/db/encoding"
)

// Server is the GRPC SporeDB endpoint.
type Server struct {
	DB     *db.DB
	Listen string
}

// Get gets a value from the database.
func (s *Server) Get(ctx context.Context, key *api.Key) (*api.Value, error) {
	value, version, err := s.DB.Get(key.Key)
	return &api.Value{
		Version: version,
		Data:    value,
	}, err
}

// Members returns the members of a specific set.
func (s *Server) Members(ctx context.Context, key *api.Key) (*api.Values, error) {
	value, version, err := s.DB.Get(key.Key)
	if err != nil {
		return nil, err
	}

	set := encoding.NewSet()
	err = set.UnmarshalBinary(value)
	if err != nil {
		return nil, err
	}

	values := &api.Values{
		Version: version,
	}

	for key := range set.Elements {
		values.Data = append(values.Data, []byte(key))
	}
	return values, nil
}

// Contains returns whether a particular set contains a specific value or not.
func (s *Server) Contains(ctx context.Context, kv *api.KeyValue) (*api.Boolean, error) {
	value, _, err := s.DB.Get(kv.Key)
	if err != nil {
		return nil, err
	}

	set := encoding.NewSet()
	err = set.UnmarshalBinary(value)
	if err != nil {
		return nil, err
	}

	return &api.Boolean{Boolean: set.Contains(kv.Value)}, nil
}

// Submit submits a set of operations to the database.
func (s *Server) Submit(ctx context.Context, tx *api.Transaction) (*api.Receipt, error) {
	spore := db.NewSpore()
	spore.Policy = tx.Policy
	spore.Requirements = tx.Requirements
	spore.Operations = tx.Operations
	spore.SetTimeout(5 * time.Second)

	return &api.Receipt{Uuid: spore.Uuid}, s.DB.Submit(spore)
}

// Serve starts the SporeDB GRPC server for clients.
func (s *Server) Serve() error {
	lis, err := net.Listen("tcp", s.Listen)
	if err != nil {
		return err
	}

	srv := grpc.NewServer()
	api.RegisterSporeDBServer(srv, s)
	return srv.Serve(lis)
}
