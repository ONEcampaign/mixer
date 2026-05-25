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

package sqlquery

import (
	"context"
	"testing"

	"github.com/datacommonsorg/mixer/internal/sqldb"
	"github.com/datacommonsorg/mixer/internal/util"
	"github.com/go-test/deep"
)

func TestVariableCoverage(t *testing.T) {
	sqlClient, err := sqldb.NewSQLiteClient("../../../test/sqlquery/variable_coverage/datacommons.db")
	if err != nil {
		t.Fatalf("Could not open testing database: %v", err)
	}
	err = sqlClient.ValidateDatabase()
	if err != nil {
		t.Fatalf("SQL database validation failed: %v", err)
	}

	// Fixture:
	// var1: country/KEN [2010,2015], country/USA [2012,2020]; value='' row excluded for var2
	// var2: country/KEN [2005,2005] (the 2008 row has value='', excluded)
	// var3: country/MEX [2015-03,2018] (mixed granularity)
	// var4: entity="" rows [2000,2010] plus country/BRA [2005,2005]
	//       tests the empty-entity key-collision fix: the envelope must union
	//       all rows (including entity="") but no {E:"",V:"var4"} entry must
	//       exist distinct from the envelope key {V:"var4"}.
	want := map[util.EntityVariable]DateRange{
		// per-entity ranges for var1
		{E: "country/KEN", V: "var1"}: {Earliest: "2010", Latest: "2015"},
		{E: "country/USA", V: "var1"}: {Earliest: "2012", Latest: "2020"},
		// envelope for var1 spans both entities
		{V: "var1"}: {Earliest: "2010", Latest: "2020"},
		// per-entity range for var2 (value='' row excluded)
		{E: "country/KEN", V: "var2"}: {Earliest: "2005", Latest: "2005"},
		// var2 present at only one entity: envelope == per-entity range
		{V: "var2"}: {Earliest: "2005", Latest: "2005"},
		// per-entity range for var3 (mixed granularity)
		{E: "country/MEX", V: "var3"}: {Earliest: "2015-03", Latest: "2018"},
		// var3 present at only one entity: envelope == per-entity range
		{V: "var3"}: {Earliest: "2015-03", Latest: "2018"},
		// var4: only the real entity gets a per-entity key; the envelope must
		// union the empty-entity rows' dates [2000,2010] with BRA [2005,2005].
		{E: "country/BRA", V: "var4"}: {Earliest: "2005", Latest: "2005"},
		{V: "var4"}: {Earliest: "2000", Latest: "2010"},
	}

	got, err := VariableCoverage(context.Background(), sqlClient)
	if err != nil {
		t.Fatalf("VariableCoverage returned error: %v", err)
	}

	if diff := deep.Equal(got, want); diff != nil {
		t.Errorf("Unexpected diff: %v", diff)
	}

	// Verify the collision invariant explicitly: there must be no
	// {E:"",V:"var4"} entry separate from the envelope key {V:"var4"}.
	// Before the fix, a row with entity="" would write to {E:"",V:"var4"},
	// which is the same map key as {V:"var4"} (zero value of E), and could
	// clobber the accumulated envelope.
	if _, ok := got[util.EntityVariable{E: "", V: "var4"}]; ok {
		// This key exists — but it is the envelope key (E=="" is the zero
		// value), so check it equals the expected envelope rather than a
		// stale single-row value that would indicate the bug regressed.
		envelope := got[util.EntityVariable{V: "var4"}]
		collision := got[util.EntityVariable{E: "", V: "var4"}]
		if collision != envelope {
			t.Errorf("empty-entity collision detected: {E:\"\",V:\"var4\"}=%v but envelope {V:\"var4\"}=%v", collision, envelope)
		}
	}
}
