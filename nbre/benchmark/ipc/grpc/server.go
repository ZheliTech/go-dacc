// Copyright (C) 2018 go-dacc authors
//
// This file is part of the go-dacc library.
//
// the go-dacc library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dacc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dacc library.  If not, see <http://www.gnu.org/licenses/>.
//

package main

import (
	"log"
	"net"

	"github.com/daccproject/go-dacc/nbre/benchmark/ipc/grpc/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	IPCAddress = "127.0.0.1:8696"
)

func main() {
	server := NewServer()

	listener, err := net.Listen("tcp", IPCAddress)
	if err != nil {
		panic("Failed to listen " + IPCAddress + " for ipc server")
	}

	if err := server.Serve(listener); err != nil {
		panic("Failed to start ipc server")
	}
	server.Stop()
	log.Println("GRPC server stoped.")
}

func NewServer() *grpc.Server {
	maxSize := 64 * 1024 * 1024
	rpc := grpc.NewServer(grpc.MaxRecvMsgSize(maxSize))
	server := &BenchmarkService{}
	ipcpb.RegisterBenchmarkServiceServer(rpc, server)
	return rpc
}

type BenchmarkService struct {
}

func (s *BenchmarkService) Transfer(ctx context.Context, req *ipcpb.Benchmark) (*ipcpb.Benchmark, error) {
	return &ipcpb.Benchmark{Data: req.Data}, nil
}