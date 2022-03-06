package extendedgorm

import "testing"

func Test_ExtendedDB_Connect(t *testing.T) {
	extendedDB := &ExtendedDB{}

	err := extendedDB.Connect("TEST")

	if err != nil {
		t.Error(err)
	}
}
