package patchpanel

import (
	"errors"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"
)

type TestStruct struct {
	Port     int           `default:"1357"`
	Greeting string        `friendly:"howdy"`
	MaxWait  time.Duration `default:"5m"`
	// wrapped int
	BestMonth               time.Month `default:"11"` // november
	BackToTheFuture         time.Time  `dest:"1985-10-26T09:00:00-07:00"`
	KitchenClock            time.Time  `dest:"3:00PM" timeFormat:"Kitchen"`
	KitchenClockSaidOutloud time.Time  `dest:"three thirty-ish" timeFormat:"Kitchen"`
	UnknownFormat           time.Time  `dest:"4:00PM" timeFormat:"zzz_unknown"`
}

func (ts TestStruct) parsedBTTFTime() time.Time {
	t, err := time.Parse(time.RFC3339, "1985-10-26T09:00:00-07:00")
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error())
	}
	return t
}

func (ts TestStruct) parsedKitchenTime() time.Time {
	t, err := time.Parse(time.Kitchen, "3:00PM")
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error())
	}
	return t
}

func TestFieldNameById(t *testing.T) {

	ts := ToReflectType(TestStruct{})

	type args struct {
		obj reflect.Type
		idx int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test field exists",
			args: args{
				obj: ts,
				idx: 0,
			},
			want: "Port",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FieldNameById(tt.args.obj, tt.args.idx); got != tt.want {
				t.Errorf("FieldNameById() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getFieldTag(t *testing.T) {

	ts := TestStruct{}
	var ConfigStruct = ToReflectType(ts)

	pp := NewPatchPanel(TokenSeparator, KeyValueSeparator)

	pp.AddParser(reflect.TypeOf(time.November), func(value string, parserHints map[string]any) (any, error) {

		monthInt, err := strconv.Atoi(value)
		if err != nil {
			return nil, errors.New("invalid value for month, expected to convert string to int")
		}

		toMonth := time.Month(monthInt)
		if toMonth > time.December {
			return nil, errors.New("month out of range")
		}

		return toMonth, nil
	})

	type args struct {
		fieldName string
		tagName   string
		t         reflect.Type
	}
	tests := []struct {
		name               string
		args               args
		want               reflect.StructField
		want1              any
		wantErr            bool
		parserHintLocation string
	}{
		{
			name:    "nil guardrail",
			args:    args{},
			want:    reflect.StructField{},
			want1:   nil,
			wantErr: true,
		},
		{
			name: "incorrect reflect type",
			args: args{
				fieldName: "okay",
				tagName:   ":)",
				// if not checked, passing in a non-struct will cause a panic
				t: ToReflectType(time.Month(1)),
			},
			want:    reflect.StructField{},
			want1:   nil,
			wantErr: true,
		},

		{
			name: "get int",
			args: args{
				fieldName: "Port",
				tagName:   "default",
				t:         ConfigStruct,
			},
			want: reflect.StructField{
				Name: "Port",
				Type: ToReflectType(0),
				Tag:  `default:"1357"`,
			},
			want1:   1357,
			wantErr: false,
		},
		{
			name: "get string",
			args: args{
				fieldName: "Greeting",
				tagName:   "friendly",
				t:         ConfigStruct,
			},
			want: reflect.StructField{
				Name: "Greeting",
				Type: ToReflectType(""),
				Tag:  `friendly:"howdy"`,
			},
			want1:   "howdy",
			wantErr: false,
		},
		{
			name: "get duration",
			args: args{
				fieldName: "MaxWait",
				tagName:   "default",
				t:         ConfigStruct,
			},
			want: reflect.StructField{
				Name: "MaxWait",
				Type: ToReflectType(time.Duration(0)),
				Tag:  `default:"5m"`,
			},
			want1:   5 * time.Minute,
			wantErr: false,
		},
		{
			name: "get month using custom parser",
			args: args{
				fieldName: "BestMonth",
				tagName:   "default",
				t:         ConfigStruct,
			},
			want: reflect.StructField{
				Name: "BestMonth",
				Type: ToReflectType(time.Month(1)),
				Tag:  `default:"11"`,
			},
			want1:   time.November,
			wantErr: false,
		},
		{
			name: "get time.Time",
			args: args{
				fieldName: "BackToTheFuture",
				tagName:   "dest",
				t:         ConfigStruct,
			},
			want: reflect.StructField{
				Name: "BackToTheFuture",
				Type: ToReflectType(time.Time{}),
				Tag:  `dest:"1985-10-26T09:00:00-07:00"`,
			},
			want1:   ts.parsedBTTFTime(),
			wantErr: false,
		},
		{
			name: "get time.Time with parser hint",
			args: args{
				fieldName: "KitchenClock",
				tagName:   "dest",
				t:         ConfigStruct,
			},
			want: reflect.StructField{
				Name: "KitchenClock",
				Type: ToReflectType(time.Time{}),
				// note that parser hints are included
				Tag: `dest:"3:00PM" timeFormat:"Kitchen"`,
			},
			want1:              ts.parsedKitchenTime(),
			wantErr:            false,
			parserHintLocation: "timeFormat",
		},
		{
			name: "get time.Time with parser hint for known format, but invalid input",
			args: args{
				fieldName: "KitchenClockSaidOutloud",
				tagName:   "dest",
				t:         ConfigStruct,
			},
			want: reflect.StructField{
				Name: "KitchenClockSaidOutloud",
				Type: ToReflectType(time.Time{}),
				// note that parser hints are included
				Tag: `dest:"three thirty-ish" timeFormat:"Kitchen"`,
			},
			want1:              time.Time{},
			wantErr:            true,
			parserHintLocation: "timeFormat",
		},
		{
			name: "get time.Time with unknown format",
			args: args{
				fieldName: "UnknownFormat",
				tagName:   "dest",
				t:         ConfigStruct,
			},
			want: reflect.StructField{
				Name: "UnknownFormat",
				Type: ToReflectType(time.Time{}),
				// note that parser hints are included
				Tag: `dest:"4:00PM" timeFormat:"zzz_unknown"`,
			},
			want1:              time.Time{},
			wantErr:            true,
			parserHintLocation: "timeFormat",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := pp.GetFieldTag(tt.args.fieldName, tt.args.tagName, tt.args.t, []string{tt.parserHintLocation})
			if (err != nil) != tt.wantErr {
				t.Errorf("getFieldTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// we don't want to do a DeepEqual as we don't particularly care about offsets
			// while we could branch for int size based on platform architectures (e.g. 4 v 8 for 32 v 64 bit),
			// we'd still need to count offsets for variably sized payloads like strings.
			if got.Name != tt.want.Name {
				t.Errorf("getFieldTag() got.Name = %v, want %v", got.Name, tt.want.Name)
			}

			if got.Type != tt.want.Type {
				t.Errorf("getFieldTag() got.Type = %v, want %v", got.Type, tt.want.Type)
			}

			if got.Tag != tt.want.Tag {
				t.Errorf("getFieldTag() got.Tag = %v, want %v", got.Tag, tt.want.Tag)
			}

			if got1 != tt.want1 {
				t.Errorf("getFieldTag() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
