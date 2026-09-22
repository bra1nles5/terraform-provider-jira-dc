package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestKeepLeadCase(t *testing.T) {
	tests := []struct {
		name     string
		known    types.String
		fromJira string
		want     types.String
	}{
		{"same spelling", types.StringValue("jsmith"), "jsmith", types.StringValue("jsmith")},
		{"other case keeps configuration", types.StringValue("jsmith"), "JSmith", types.StringValue("jsmith")},
		{"another user comes from Jira", types.StringValue("jsmith"), "adoe", types.StringValue("adoe")},
		{"null after import comes from Jira", types.StringNull(), "JSmith", types.StringValue("JSmith")},
		{"unknown comes from Jira", types.StringUnknown(), "JSmith", types.StringValue("JSmith")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := keepLeadCase(tt.known, tt.fromJira); !got.Equal(tt.want) {
				t.Errorf("keepLeadCase(%s, %q) = %s, want %s", tt.known, tt.fromJira, got, tt.want)
			}
		})
	}
}
