// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package node

import (
	"context"
	"encoding/json"

	cbt "cloud.google.com/go/bigtable"
	"github.com/datacommonsorg/mixer/internal/server/model"
	"github.com/datacommonsorg/mixer/internal/store/bigtable"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/datacommonsorg/mixer/internal/store"
	"github.com/datacommonsorg/mixer/internal/util"

	pb "github.com/datacommonsorg/mixer/internal/proto"
)

// GetPropertyValues implements API for Mixer.GetPropertyValues.
func GetPropertyValues(ctx context.Context,
	in *pb.GetPropertyValuesRequest, store *store.Store) (*pb.GetPropertyValuesResponse, error) {
	dcids := in.GetDcids()
	prop := in.GetProperty()
	typ := in.GetValueType()
	direction := in.GetDirection()
	limit := int(in.GetLimit())

	// Check arguments
	if prop == "" || len(dcids) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Missing required arguments")
	}
	if !util.CheckValidDCIDs(dcids) {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid DCIDs")
	}

	// Get in, out or both direction
	var (
		inArc  = true
		outArc = true
		inRes  = map[string][]*model.Node{}
		outRes = map[string][]*model.Node{}
	)
	var err error
	if direction == "in" {
		outArc = false
	} else if direction == "out" {
		inArc = false
	}

	if inArc {
		inRes, err = GetPropertyValuesHelper(ctx, store, dcids, prop, false)
		if err != nil {
			return nil, err
		}
	}
	if outArc {
		outRes, err = GetPropertyValuesHelper(ctx, store, dcids, prop, true)
		if err != nil {
			return nil, err
		}
	}

	result := make(map[string]map[string][]*model.Node)
	for _, dcid := range dcids {
		result[dcid] = map[string][]*model.Node{}
	}
	for dcid, nodes := range inRes {
		trimedNodes := trimNodes(nodes, typ, limit)
		if len(trimedNodes) > 0 {
			result[dcid]["in"] = trimedNodes

		}
	}
	for dcid, nodes := range outRes {
		trimedNodes := trimNodes(nodes, typ, limit)
		if len(trimedNodes) > 0 {
			result[dcid]["out"] = trimedNodes
		}
	}

	jsonRaw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetPropertyValuesResponse{Payload: string(jsonRaw)}, nil
}

// GetPropertyValuesHelper get property values.
func GetPropertyValuesHelper(
	ctx context.Context,
	store *store.Store,
	dcids []string,
	prop string,
	arcOut bool,
) (map[string][]*model.Node, error) {
	rowList := bigtable.BuildPropertyValuesKey(dcids, prop, arcOut)
	// Current branch cache is targeted on new stats (without addition of schema etc),
	// so only use base cache data for property value.
	//
	// TODO(shifucun): perform a systematic check on current cache data and see
	// if this is still true.
	return readPropertyValues(ctx, store, rowList)
}

func trimNodes(nodes []*model.Node, typ string, limit int) []*model.Node {
	if limit == 0 && typ == "" {
		return nodes
	}
	result := []*model.Node{}
	for _, node := range nodes {
		if typ == "" {
			result = append(result, node)
		} else {
			for _, t := range node.Types {
				if t == typ {
					result = append(result, node)
					break
				}
			}
		}
		if limit > 0 && len(result) == limit {
			break
		}
	}
	return result
}

func readPropertyValues(
	ctx context.Context,
	store *store.Store,
	rowList cbt.RowList,
) (map[string][]*model.Node, error) {
	// Only read property value from base cache.
	// Branch cache only contains supplement data but not other properties yet.
	baseDataMap, _, err := bigtable.Read(
		ctx,
		store.BtGroup,
		rowList,
		func(dcid string, jsonRaw []byte) (interface{}, error) {
			var propVals model.PropValueCache
			err := json.Unmarshal(jsonRaw, &propVals)
			if err != nil {
				return nil, err
			}
			return propVals.Nodes, nil
		},
		nil,
		false, /* readBranch */
	)
	if err != nil {
		return nil, err
	}
	result := map[string][]*model.Node{}
	for dcid, data := range baseDataMap {
		if data != nil {
			result[dcid] = data.([]*model.Node)
		}
	}
	return result, nil
}
