package main

import "testing"

func TestQuote(t *testing.T) {
	mscs := getMSCs("> <@foo:matrix.org> MSC2444 just got merged\n\nwew lad; msc2444 has been a long time coming")
	if len(mscs) != 1 {
		t.Fail()
	} else if mscs[0] != 2444 {
		t.Fail()
	}
}

func TestCapitalization(t *testing.T) {
	mscs := getMSCs("msc123 foo bar baz MSC234\nfoo bar MsC345 MSC456")
	if len(mscs) != 3 {
		t.Fail()
	} else if mscs[0] != 123 {
		t.Fail()
	} else if mscs[1] != 234 {
		t.Fail()
	} else if mscs[2] != 456 {
		t.Fail()
	}
}

func TestNoRepeats(t *testing.T) {
	mscs := getMSCs("That msc seems to be the last prerequisite for MSC2677: https://github.com/uhoreg/matrix-doc/blob/aggregations-reactions/proposals/2677-reactions.md#msc2677-annotations-and-reactions")
	if len(mscs) != 1 {
		t.Fail()
	} else if mscs[0] != 2677 {
		t.Fail()
	}
}
