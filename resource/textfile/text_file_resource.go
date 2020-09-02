package textfile

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/numtide/terraform-provider-linuxbox/sshsession"
	"github.com/pkg/errors"
)

func Resource() *schema.Resource {
	return &schema.Resource{
		Create: resourceUpdateAndCreate,
		Read:   resourceRead,
		Update: resourceUpdateAndCreate,
		Delete: resourceDelete,

		Schema: map[string]*schema.Schema{
			"ssh_key": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},

			"ssh_user": &schema.Schema{
				Type:     schema.TypeString,
				Required: false,
				Default:  "root",
				Optional: true,
			},

			"host_address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"path": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"content": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"owner": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "root",
			},

			"group": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "root",
			},

			"mode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "755",
			},
		},
	}
}

func resourceUpdateAndCreate(d *schema.ResourceData, m interface{}) error {

	content := []byte(d.Get("content").(string))

	path := d.Get("path").(string)

	owner := d.Get("owner").(string)
	group := d.Get("group").(string)

	mode := d.Get("mode").(string)

	cmd := fmt.Sprintf("echo '%s' | base64 -d | cat > '%s' && chown '%s':'%s' '%s' && chmod '%s' '%s'", base64.StdEncoding.EncodeToString(content), path, owner, group, path, mode, path)
	stdout, stderr, err := sshsession.Run(d, cmd)
	if err != nil {
		return errors.Wrapf(err, "error while creating file %s:\nSTDOUT:\n%s\nSTDERR:\n%s\n", path, string(stdout), string(stderr))
	}

	sh := sha256.New()

	sh.Write(content)
	sum := sh.Sum(nil)

	d.SetId(hex.EncodeToString(sum[:]))

	return nil
}

func resourceRead(d *schema.ResourceData, m interface{}) error {
	path := d.Get("path").(string)

	{
		cmd := fmt.Sprintf("stat -c '%%U %%G %%a' '%s'", path)

		stdout, _, err := sshsession.Run(d, cmd)
		if err != nil {
			d.SetId("")
			return nil
		}

		stdoutString := string(stdout)

		stdoutString = strings.TrimSuffix(stdoutString, "\n")

		parts := strings.Split(stdoutString, " ")
		if len(parts) != 3 {
			return errors.Errorf("malformed output of %q: %q", cmd, stdoutString)
		}

		d.Set("owner", parts[0])
		d.Set("group", parts[1])
		d.Set("mode", parts[2])
	}

	{

		cmd := fmt.Sprintf("cat '%s'", path)
		stdout, _, err := sshsession.Run(d, cmd)
		if err != nil {
			return errors.Wrapf(err, "while getting content of %s", path)
		}

		stdoutString := string(stdout)

		d.Set("content", stdoutString)

	}

	return nil

}

func resourceDelete(d *schema.ResourceData, m interface{}) error {
	path := d.Get("path").(string)

	cmd := fmt.Sprintf("rm -f '%s'", path)

	stdout, stderr, err := sshsession.Run(d, cmd)
	if err != nil {
		return errors.Wrapf(err, "error while deletin file %s:\nSTDOUT:\n%s\nSTDERR:\n%s\n", path, string(stdout), string(stderr))
	}

	return nil
}