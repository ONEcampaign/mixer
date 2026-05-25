// Copyright 2025 Google LLC
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

package server

import (
	"context"
	"testing"

	pb "github.com/datacommonsorg/mixer/internal/proto"
	"github.com/datacommonsorg/mixer/internal/server/cache"
	"github.com/datacommonsorg/mixer/internal/sqldb/sqlquery"
	"github.com/datacommonsorg/mixer/internal/util"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestV2VariableCoverage(t *testing.T) {
	// Build a coverage map exercising the envelope and per-entity paths.
	// var1: KEN [2010,2015], USA [2012,2020]; envelope [2010,2020]
	// var2: KEN [2005,2005]; envelope [2005,2005]
	// var3: no entry (absent → omitted)
	coverageMap := map[util.EntityVariable]sqlquery.DateRange{
		{E: "country/KEN", V: "var1"}: {Earliest: "2010", Latest: "2015"},
		{E: "country/USA", V: "var1"}: {Earliest: "2012", Latest: "2020"},
		{V: "var1"}:                   {Earliest: "2010", Latest: "2020"},
		{E: "country/KEN", V: "var2"}: {Earliest: "2005", Latest: "2005"},
		{V: "var2"}:                   {Earliest: "2005", Latest: "2005"},
	}

	s := &Server{}
	s.cachedata.Store(cache.NewCoverageCache(coverageMap))

	ctx := context.Background()

	for _, tc := range []struct {
		desc     string
		in       *pb.VariableCoverageRequest
		wantResp *pb.VariableCoverageResponse
	}{
		{
			desc: "placeless: both present vars returned; absent var omitted",
			in: &pb.VariableCoverageRequest{
				Variables: []string{"var1", "var2", "var3"},
			},
			wantResp: &pb.VariableCoverageResponse{
				VariableCoverage: map[string]*pb.DateRange{
					"var1": {Earliest: "2010", Latest: "2020"},
					"var2": {Earliest: "2005", Latest: "2005"},
				},
			},
		},
		{
			desc: "with entities: entity coverage populated for present pairs",
			in: &pb.VariableCoverageRequest{
				Variables: []string{"var1", "var2", "var3"},
				Entities:  []string{"country/KEN", "country/USA"},
			},
			wantResp: &pb.VariableCoverageResponse{
				VariableCoverage: map[string]*pb.DateRange{
					"var1": {Earliest: "2010", Latest: "2020"},
					"var2": {Earliest: "2005", Latest: "2005"},
				},
				EntityCoverage: map[string]*pb.EntityRanges{
					"var1": {
						Entity: map[string]*pb.DateRange{
							"country/KEN": {Earliest: "2010", Latest: "2015"},
							"country/USA": {Earliest: "2012", Latest: "2020"},
						},
					},
					"var2": {
						Entity: map[string]*pb.DateRange{
							"country/KEN": {Earliest: "2005", Latest: "2005"},
							// country/USA not in map for var2 → omitted
						},
					},
					// var3 absent from map → no entity_coverage entry
				},
			},
		},
		{
			desc: "empty variables: empty response",
			in:   &pb.VariableCoverageRequest{},
			wantResp: &pb.VariableCoverageResponse{},
		},
		{
			desc: "absent variable only: empty response",
			in: &pb.VariableCoverageRequest{
				Variables: []string{"var3"},
				Entities:  []string{"country/KEN"},
			},
			wantResp: &pb.VariableCoverageResponse{},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := s.V2VariableCoverage(ctx, tc.in)
			if err != nil {
				t.Fatalf("V2VariableCoverage returned error: %v", err)
			}
			if diff := cmp.Diff(tc.wantResp, got, protocmp.Transform()); diff != "" {
				t.Errorf("unexpected response diff (-want +got):\n%s", diff)
			}
		})
	}
}
