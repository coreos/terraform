package localfile

import (
	"fmt"
	"testing"

	"io/ioutil"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestLocalFile_Basic(t *testing.T) {
	var cases = []struct {
		path    string
		content string
		config  string
	}{
		{
			"localfile",
			"This is some content",
			`resource "localfile_file" "file" {
         content     = "This is some content"
         destination = "localfile"
      }`,
		},
	}

	for _, tt := range cases {
		r.UnitTest(t, r.TestCase{
			Providers: testProviders,
			Steps: []r.TestStep{
				{
					Config: tt.config,
					Check: func(s *terraform.State) error {
						content, err := ioutil.ReadFile(tt.path)
						if err != nil {
							return fmt.Errorf("config:\n%s\n,got: %s\n", tt.config, err)
						}
						if string(content) != tt.content {
							return fmt.Errorf("config:\n%s\ngot:\n%s\nwant:\n%s\n", tt.config, content, tt.content)
						}
						return nil
					},
				},
			},
		})
	}
}
