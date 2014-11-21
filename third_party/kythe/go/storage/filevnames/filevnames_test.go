package filevnames

import (
	"strings"
	"testing"

	spb "third_party/kythe/proto/storage_proto"

	"code.google.com/p/goprotobuf/proto"
)

// Copied exactly from kythe/javatests/com/google/devtools/kythe/extractors/shared/FileVNamesTest.java
// This could be prettier in Go but a copy ensures better compatibility between
// the libraries.
var testConfig = strings.Join([]string{
	"[",
	"  {",
	"    \"pattern\": \"static/path\",",
	"    \"vname\": {",
	"      \"root\": \"root\",",
	"      \"corpus\": \"static\"",
	"    }",
	"  },",
	"  {",
	"    \"pattern\": \"dup/path\",",
	"    \"vname\": {",
	"      \"corpus\": \"first\"",
	"    }",
	"  },",
	"  {",
	"    \"pattern\": \"dup/path2\",",
	"    \"vname\": {",
	"      \"corpus\": \"second\"",
	"    }",
	"  },",
	"  {",
	"    \"pattern\": \"(grp1)/(\\\\d+)/(.*)\",",
	"    \"vname\": {",
	"      \"root\": \"@2@\",",
	"      \"corpus\": \"@1@/@3@\"",
	"    }",
	"  },",
	"  {",
	"    \"pattern\": \"campfire-out/[^/]+/([^/]+)/java/.*[.]jar!/.*\",",
	"    \"vname\": {",
	"      \"root\": \"java\",",
	"      \"corpus\": \"@1@\"",
	"    }",
	"  },",
	"  {",
	"    \"pattern\": \"third_party/([^/]+)/.*[.]jar!/.*\",",
	"    \"vname\": {",
	"      \"root\": \"@1@\",",
	"      \"corpus\": \"third_party\"",
	"    }",
	"  },",
	"  {",
	"    \"pattern\": \"([^/]+)/java/.*\",",
	"    \"vname\": {",
	"      \"root\": \"java\",",
	"      \"corpus\": \"@1@\"",
	"    }",
	"  },",
	"  {",
	"    \"pattern\": \"([^/]+)/.*\",",
	"    \"vname\": {",
	"      \"corpus\": \"@1@\"",
	"    }",
	"  }",
	"]"}, "\n")

func TestParsing(t *testing.T) {
	c, err := ParseJSON([]byte(testConfig))
	if err != nil {
		t.Error(err)
	} else if c == nil {
		t.Error("nil Config")
	}
}

func TestLookup(t *testing.T) {
	tests := []struct {
		path     string
		expected *spb.VName
	}{
		// default lookup
		{"", &spb.VName{}},

		// static
		{"static/path", corpusRoot("static", "root")},

		// ordered
		{"dup/path", corpus("first")},
		{"dup/path2", corpus("second")},

		// groups
		{"corpus/some/path/here", corpus("corpus")},
		{"grp1/12345/endingGroup", corpusRoot("grp1/endingGroup", "12345")},
		{"campfire-out/bin/kythe/java/some/path/A.jar!/some/path/A.class", corpusRoot("kythe", "java")},
		{"third_party/kythe/java/com/google/devtools/kythe/util/KytheURI.java", corpusRoot("kythe", "java")},
		{"otherCorpus/java/com/google/devtools/kythe/util/KytheURI.java", corpusRoot("otherCorpus", "java")},
	}

	c, _ := ParseJSON([]byte(testConfig))

	for _, test := range tests {
		res := c.LookupVName(test.path)
		if !proto.Equal(res, test.expected) {
			t.Errorf("For path %q: expected {%+v}; got {%+v}", test.path, test.expected, res)
		}
	}
}

func corpus(corpus string) *spb.VName {
	return corpusRoot(corpus, "")
}

func corpusRoot(corpus, root string) *spb.VName {
	return corpusPath(corpus, root, "")
}

func corpusPath(corpus, root, path string) *spb.VName {
	return &spb.VName{
		Corpus: &corpus,
		Root:   &root,
		Path:   &path,
	}
}
