package docker

import (
    "reflect"
    "testing"
)

func TestLinesWithTrailingLineBreak(t *testing.T) {
    input := "This is a multiline string.\nIt spans multiple lines.\n"
    expected := []string {
        "This is a multiline string.",
        "It spans multiple lines.",
    }
    actual := Lines(input)

    if !reflect.DeepEqual(expected, actual) {
        t.Fatalf("expected '%v', actual '%v'", expected, actual)
    }
}
