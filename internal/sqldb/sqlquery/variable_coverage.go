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

	"github.com/datacommonsorg/mixer/internal/sqldb"
	"github.com/datacommonsorg/mixer/internal/util"
)

// DateRange is the earliest/latest observed date for a coverage key.
type DateRange struct{ Earliest, Latest string }

// minStr returns the lexicographically smaller of a and b, treating "" as
// "no bound yet" (seeds to the first non-empty value).
func minStr(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if a < b {
		return a
	}
	return b
}

// maxStr returns the lexicographically larger of a and b, treating "" as
// "no bound yet".
func maxStr(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if a > b {
		return a
	}
	return b
}

// VariableCoverage builds the dual-keyed coverage map: {E,V}->range plus a
// folded {V}->envelope (E==""), one pass over the GROUP BY rows.
func VariableCoverage(ctx context.Context, sqlClient *sqldb.SQLClient) (map[util.EntityVariable]DateRange, error) {
	rows, err := sqlClient.GetVariableCoverageRows(ctx)
	if err != nil {
		return nil, err
	}
	m := map[util.EntityVariable]DateRange{}
	for _, row := range rows {
		e, v, minD, maxD := row.Entity, row.Variable, row.MinDate, row.MaxDate
		// Per-entity range. Skip e=="" to avoid a key collision: the envelope
		// is stored at {E:"",V:v} (zero value of E), so writing a per-entity
		// entry for an empty-entity row would silently overwrite it, causing
		// later folds to lose previously accumulated date bounds.
		if e != "" {
			m[util.EntityVariable{E: e, V: v}] = DateRange{Earliest: minD, Latest: maxD}
		}
		// Fold into the variable-level envelope (E=="") — always, so that
		// even empty-entity rows contribute their date range to the union.
		env := m[util.EntityVariable{V: v}]
		env.Earliest = minStr(env.Earliest, minD)
		env.Latest = maxStr(env.Latest, maxD)
		m[util.EntityVariable{V: v}] = env
	}
	return m, nil
}
