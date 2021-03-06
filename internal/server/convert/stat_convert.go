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

package convert

import (
	pb "github.com/datacommonsorg/mixer/internal/proto"
	"github.com/datacommonsorg/mixer/internal/server/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

// UnitConversion represents conversion sepc for units.
type UnitConversion struct {
	Unit    string
	Scaling float64
}

// UnitMapping maps unit schemas with scaling factor.
var UnitMapping = map[string]*UnitConversion{
	"GigawattHour": {
		Unit:    "KilowattHour",
		Scaling: 1000000,
	},
}

// ToObsSeriesPb converts ChartStore to pb.ObsTimeSerie
func ToObsSeriesPb(token string, jsonRaw []byte) (
	interface{}, error) {
	pbData := &pb.ChartStore{}
	if err := protojson.Unmarshal(jsonRaw, pbData); err != nil {
		return nil, err
	}
	switch x := pbData.Val.(type) {
	case *pb.ChartStore_ObsTimeSeries:
		x.ObsTimeSeries.PlaceName = ""
		ret := x.ObsTimeSeries
		// Unify unit.
		for _, series := range ret.SourceSeries {
			if conversion, ok := UnitMapping[series.Unit]; ok {
				series.Unit = conversion.Unit
				for date := range series.Val {
					series.Val[date] *= conversion.Scaling
				}
			}
		}
		return ret, nil
	case nil:
		return nil, status.Error(codes.NotFound, "ChartStore.Val is not set")
	default:
		return nil, status.Errorf(codes.NotFound,
			"ChartStore.Val has unexpected type %T", x)
	}
}

// ToObsSeries converts ChartStore to ObsSeries
func ToObsSeries(token string, jsonRaw []byte) (
	interface{}, error) {
	pbData := &pb.ChartStore{}
	if err := protojson.Unmarshal(jsonRaw, pbData); err != nil {
		return nil, err
	}
	switch x := pbData.Val.(type) {
	case *pb.ChartStore_ObsTimeSeries:
		pbSourceSeries := x.ObsTimeSeries.GetSourceSeries()
		ret := &model.ObsTimeSeries{
			Data:         x.ObsTimeSeries.GetData(),
			PlaceName:    x.ObsTimeSeries.GetPlaceName(),
			SourceSeries: make([]*model.SourceSeries, len(pbSourceSeries)),
		}
		for i, source := range pbSourceSeries {
			if conversion, ok := UnitMapping[source.Unit]; ok {
				source.Unit = conversion.Unit
				for date := range source.Val {
					source.Val[date] *= conversion.Scaling
				}
			}
			ret.SourceSeries[i] = &model.SourceSeries{
				ImportName:        source.GetImportName(),
				ObservationPeriod: source.GetObservationPeriod(),
				MeasurementMethod: source.GetMeasurementMethod(),
				ScalingFactor:     source.GetScalingFactor(),
				Unit:              source.GetUnit(),
				ProvenanceURL:     source.GetProvenanceUrl(),
				Val:               source.GetVal(),
			}

		}
		ret.ProvenanceURL = x.ObsTimeSeries.GetProvenanceUrl()
		return ret, nil
	case nil:
		return nil, status.Error(codes.Internal, "ChartStore.Val is not set")
	default:
		return nil, status.Errorf(codes.Internal, "ChartStore.Val has unexpected type %T", x)
	}
}

// ToObsCollection converts ChartStore to pb.ObsCollection
func ToObsCollection(token string, jsonRaw []byte) (
	interface{}, error) {
	pbData := &pb.ChartStore{}
	if err := protojson.Unmarshal(jsonRaw, pbData); err != nil {
		return nil, err
	}
	switch x := pbData.Val.(type) {
	case *pb.ChartStore_ObsCollection:
		ret := x.ObsCollection
		// Unify unit.
		for _, series := range ret.SourceCohorts {
			if conversion, ok := UnitMapping[series.Unit]; ok {
				series.Unit = conversion.Unit
				for date := range series.Val {
					series.Val[date] *= conversion.Scaling
				}
			}
		}
		return ret, nil
	case nil:
		return nil, status.Error(codes.Internal,
			"ChartStore.Val is not set")
	default:
		return nil, status.Errorf(codes.Internal,
			"ChartStore.Val has unexpected type %T", x)
	}
}
