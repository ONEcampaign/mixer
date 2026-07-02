// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package propertyvalues

import (
	"testing"

	"github.com/datacommonsorg/mixer/internal/sqldb"
	"github.com/datacommonsorg/mixer/internal/util"
)

// Regression test: on case/accent-insensitive or PAD SPACE collations
// (the MySQL / CloudSQL default), the triples join can return a stored DCID
// that differs from the requested node, so the response builder must not
// assume every returned id was pre-seeded in the response map. It used to
// panic with "assignment to entry in nil map".
func TestBuildTriplesResponseCollationVariantDcid(t *testing.T) {
	entityInfos := map[string]*entityInfo{}
	for _, tc := range []struct {
		direction string
		triple    *sqldb.Triple
	}{
		{
			util.DirectionOut,
			&sqldb.Triple{SubjectID: "Country/USA", Predicate: "name", ObjectValue: "United States"},
		},
		{
			util.DirectionIn,
			&sqldb.Triple{SubjectID: "geoId/06", Predicate: "containedInPlace", ObjectID: "Country/USA"},
		},
	} {
		// Requested node is a case variant of the DCID the join returned.
		resp := buildTriplesResponse(
			[]string{"country/usa"}, []*sqldb.Triple{tc.triple}, entityInfos, tc.direction)
		if _, ok := resp["Country/USA"][tc.triple.Predicate]; !ok {
			t.Errorf("buildTriplesResponse(%s): missing stored-DCID entry, got %v", tc.direction, resp)
		}
	}
}
