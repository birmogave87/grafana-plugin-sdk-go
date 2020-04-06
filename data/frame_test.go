package data_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

func TestFrame(t *testing.T) {
	df := data.NewFrame("http_requests_total",
		data.NewField("timestamp", nil, []time.Time{time.Now(), time.Now(), time.Now()}).SetConfig(&data.FieldConfig{
			Title: "A time Column.",
		}),
		data.NewField("value", data.Labels{"service": "auth"}, []float64{1.0, 2.0, 3.0}),
		data.NewField("category", data.Labels{"service": "auth"}, []string{"foo", "bar", "test"}),
		data.NewField("valid", data.Labels{"service": "auth"}, []bool{true, false, true}),
	)

	if df.Rows() != 3 {
		t.Fatal("unexpected length")
	}
}

func ExampleNewFrame() {
	aTime := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	var anInt64 int64 = 12
	frame := data.NewFrame("Frame Name",
		data.NewField("Time", nil, []time.Time{aTime, aTime.Add(time.Minute)}),
		data.NewField("Temp", data.Labels{"place": "Ecuador"}, []float64{1, math.NaN()}),
		data.NewField("Count", data.Labels{"place": "Ecuador"}, []*int64{&anInt64, nil}),
	)
	fmt.Println(frame.String())
	// Output:
	// Name: Frame Name
	// Dimensions: 3 Fields by 2 Rows
	// +-------------------------------+-----------------------+-----------------------+
	// | Name: Time                    | Name: Temp            | Name: Count           |
	// | Labels:                       | Labels: place=Ecuador | Labels: place=Ecuador |
	// | Type: []time.Time             | Type: []float64       | Type: []*int64        |
	// +-------------------------------+-----------------------+-----------------------+
	// | 2020-01-02 03:04:05 +0000 UTC | 1                     | 12                    |
	// | 2020-01-02 03:05:05 +0000 UTC | NaN                   | null                  |
	// +-------------------------------+-----------------------+-----------------------+
}

func TestStringTable(t *testing.T) {
	frame := data.NewFrame("sTest",
		data.NewField("", nil, make([]bool, 3)),
		data.NewField("", nil, make([]bool, 3)),
		data.NewField("", nil, make([]bool, 3)),
	)
	tests := []struct {
		name      string
		maxWidth  int
		maxLength int
		output    string
	}{
		{
			name:      "at max width and length",
			maxWidth:  3,
			maxLength: 3,
			output: `Name: sTest
Dimensions: 3 Fields by 3 Rows
+--------------+--------------+--------------+
| Name:        | Name:        | Name:        |
| Labels:      | Labels:      | Labels:      |
| Type: []bool | Type: []bool | Type: []bool |
+--------------+--------------+--------------+
| false        | false        | false        |
| false        | false        | false        |
| false        | false        | false        |
+--------------+--------------+--------------+
`,
		},
		{
			name:      "above max width and length",
			maxWidth:  2,
			maxLength: 2,
			output: `Name: sTest
Dimensions: 3 Fields by 3 Rows
+--------------+----------------+
| Name:        | ...+2 field... |
| Labels:      |                |
| Type: []bool |                |
+--------------+----------------+
| false        | ...            |
| ...          | ...            |
+--------------+----------------+
`,
		},
		{
			name:      "no length",
			maxWidth:  10,
			maxLength: 0,
			output: `Name: sTest
Dimensions: 3 Fields by 3 Rows
+--------------+--------------+--------------+
| Name:        | Name:        | Name:        |
| Labels:      | Labels:      | Labels:      |
| Type: []bool | Type: []bool | Type: []bool |
+--------------+--------------+--------------+
+--------------+--------------+--------------+
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := frame.StringTable(tt.maxWidth, tt.maxLength)
			require.NoError(t, err)
			require.Equal(t, tt.output, s)

		})
	}

}

func TestFrameWarnings(t *testing.T) {
	df := data.NewFrame("warning_test")
	df.AppendWarning("details1", "message1")
	df.AppendWarning("details2", "message2")

	if len(df.Warnings) != 2 {
		t.Fatal("expected two warnings to be appended")
	}
}

func TestDataFrameFilterRowsByField(t *testing.T) {
	tests := []struct {
		name          string
		frame         *data.Frame
		filteredFrame *data.Frame
		fieldIdx      int
		filterFunc    func(i interface{}) (bool, error)
		shouldErr     require.ErrorAssertionFunc
	}{
		{
			name: "time filter test",
			frame: data.NewFrame("time_filter_test", data.NewField("time", nil, []time.Time{
				time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
				time.Date(2020, 1, 2, 3, 4, 15, 0, time.UTC),
				time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				time.Date(2020, 1, 2, 3, 4, 45, 0, time.UTC),
			}),
				data.NewField("floats", nil, []float64{
					1.0, 2.0, 3.0, 4.0,
				})),
			filteredFrame: data.NewFrame("time_filter_test", data.NewField("time", nil, []time.Time{
				time.Date(2020, 1, 2, 3, 4, 15, 0, time.UTC),
				time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
			}),
				data.NewField("floats", nil, []float64{
					2.0, 3.0,
				})),
			fieldIdx: 0,
			filterFunc: func(i interface{}) (bool, error) {
				val, ok := i.(time.Time)
				if !ok {
					return false, fmt.Errorf("wrong type dumbface. Oh ya, stupid error even-dumber-face.")
				}
				if val.After(time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC)) && val.Before(time.Date(2020, 1, 2, 3, 4, 45, 0, time.UTC)) {
					return true, nil
				}
				return false, nil
			},
			shouldErr: require.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filteredFrame, err := tt.frame.FilterRowsByField(tt.fieldIdx, tt.filterFunc)
			tt.shouldErr(t, err)
			if diff := cmp.Diff(tt.filteredFrame, filteredFrame, data.FrameTestCompareOptions()...); diff != "" {
				t.Errorf("Result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func float32Ptr(f float32) *float32 {
	return &f
}

func float64Ptr(f float64) *float64 {
	return &f
}

func int8Ptr(i int8) *int8 {
	return &i
}

func int16Ptr(i int16) *int16 {
	return &i
}

func int32Ptr(i int32) *int32 {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}

func uint8Ptr(ui uint8) *uint8 {
	return &ui
}

func uint16Ptr(ui uint16) *uint16 {
	return &ui
}

func uint32Ptr(ui uint32) *uint32 {
	return &ui
}

func uint64Ptr(ui uint64) *uint64 {
	return &ui
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
