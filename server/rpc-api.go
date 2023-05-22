package server

import (
	_ "log"
	"net/rpc"
)

type GetArgs struct {
	Key []byte
}

type GetResult struct {
	Value []byte
}

type SetArgs struct {
	Key   []byte
	Value []byte
}

type SetResult struct {
	Success bool
}

type DelArgs struct {
	Key []byte
}

type DelResult struct {
	Success bool
}

type UpdateArgs struct {
	Key   []byte
	Value []byte
}

type UpdateResult struct {
	Success bool
}

func (srv *Server) Get(args GetArgs, result *GetResult) error {
	srv.mutex.RLock()
	value, err := srv.db.Get(args.Key)
	srv.mutex.RUnlock()

	if err != nil {
		return rpc.ErrShutdown
	}

	result.Value = value
	return nil
}

func (srv *Server) Set(args SetArgs, result *SetResult) error {
	srv.mutex.Lock()
	srv.db.Set(args.Key, args.Value)
	srv.mutex.Unlock()

	result.Success = true
	return nil
}

func (srv *Server) Del(args DelArgs, result *DelResult) error {
	srv.mutex.Lock()
	err := srv.db.Delete(args.Key)
	srv.mutex.Unlock()

	if err != nil {
		result.Success = false
		return nil
	}

	result.Success = true
	return nil
}

func (srv *Server) Update(args UpdateArgs, result *UpdateResult) error {
	srv.mutex.Lock()
	err := srv.db.Update(args.Key, args.Value)
	srv.mutex.Unlock()

	if err != nil {
		result.Success = false
		return nil
	}

	result.Success = true
	return nil
}
