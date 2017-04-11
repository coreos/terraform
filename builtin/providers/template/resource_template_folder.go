package template

import (
	"archive/tar"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/hashicorp/terraform/helper/pathorcontents"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceFolder() *schema.Resource {
	return &schema.Resource{
		Create: dataSourceTemplateFolderCreate,
		Read:   dataSourceTemplateFolderRead,
		Delete: dataSourceTemplateFolderDelete,

		Schema: map[string]*schema.Schema{
			"input_path": {
				Type:        schema.TypeString,
				Description: "Path to the folder to template",
				Required:    true,
				ForceNew:    true,
			},
			"vars": {
				Type:         schema.TypeMap,
				Optional:     true,
				Default:      make(map[string]interface{}),
				Description:  "Variables to substitute",
				ValidateFunc: validateVarsAttribute,
				ForceNew:     true,
			},
			"output_path": {
				Type:        schema.TypeString,
				Description: "Path to the output folder",
				Required:    true,
				ForceNew:    true,
			},
		},
	}
}

func dataSourceTemplateFolderRead(d *schema.ResourceData, meta interface{}) error {
	inputPath := d.Get("input_path").(string)
	outputPath := d.Get("output_path").(string)

	// If the output doesn't exist, mark the resource for creation.
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		d.SetId("")
		return nil
	}

	// If the combined hash of the input and output folders is different from the
	// stored one, mark the resource for re-creation.
	//
	// The output folder is technically enough for the general case, but by
	// hashing the input folder as well, we make development much easier: when a
	// developer modifies one of the input files, the generation is re-triggered.
	hash, err := generateID(inputPath, outputPath)
	if err != nil {
		return err
	}
	if hash != d.Id() {
		d.SetId("")
		return nil
	}

	return nil
}

func dataSourceTemplateFolderCreate(d *schema.ResourceData, meta interface{}) error {
	inputPath := d.Get("input_path").(string)
	outputPath := d.Get("output_path").(string)
	vars := d.Get("vars").(map[string]interface{})

	// Always delete the output first, otherwise files that got deleted from the
	// input folder might still be present in the output afterwards.
	if err := dataSourceTemplateFolderDelete(d, meta); err != nil {
		return err
	}

	// Recursively crawl the input files/folders and generate the output ones.
	err := filepath.Walk(inputPath, func(p string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(inputPath, p)
		return generateFolderFile(p, path.Join(outputPath, relPath), f, vars)
	})
	if err != nil {
		return err
	}

	// Compute ID.
	hash, err := generateID(inputPath, outputPath)
	if err != nil {
		return err
	}
	d.SetId(hash)

	return nil
}

func dataSourceTemplateFolderDelete(d *schema.ResourceData, _ interface{}) error {
	d.SetId("")

	outputPath := d.Get("output_path").(string)
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return nil
	}

	if err := os.RemoveAll(outputPath); err != nil {
		return fmt.Errorf("could not delete folder %q: %s", outputPath, err)
	}

	return nil
}

func generateFolderFile(inputPath, outputPath string, f os.FileInfo, vars map[string]interface{}) error {
	inputContent, _, err := pathorcontents.Read(inputPath)
	if err != nil {
		return err
	}

	outputContent, err := execute(inputContent, vars)
	if err != nil {
		return templateRenderError(fmt.Errorf("failed to render %v: %v", inputPath, err))
	}

	outputDir := path.Dir(outputPath)
	if _, err := os.Stat(outputDir); err != nil {
		if err := os.MkdirAll(outputDir, 0777); err != nil {
			return err
		}
	}

	err = ioutil.WriteFile(outputPath, []byte(outputContent), f.Mode())
	if err != nil {
		return err
	}

	return nil
}

func generateID(inputPath, outputPath string) (string, error) {
	inputHash, err := generateFolderHash(inputPath)
	if err != nil {
		return "", err
	}
	outputHash, err := generateFolderHash(outputPath)
	if err != nil {
		return "", err
	}
	checksum := sha1.Sum([]byte(inputHash + outputHash))
	return hex.EncodeToString(checksum[:]), nil
}

func generateFolderHash(folderPath string) (string, error) {
	tarData, err := tarFolder(folderPath)
	if err != nil {
		return "", fmt.Errorf("could not generate output checksum: %s", err)
	}

	checksum := sha1.Sum(tarData)
	return hex.EncodeToString(checksum[:]), nil
}

func tarFolder(folderPath string) ([]byte, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	writeFile := func(p string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		var header *tar.Header
		var file *os.File

		header, err = tar.FileInfoHeader(f, f.Name())
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(folderPath, p)
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if f.IsDir() {
			return nil
		}

		file, err = os.Open(p)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tw, file)
		return err
	}

	if err := filepath.Walk(folderPath, writeFile); err != nil {
		return []byte{}, err
	}
	if err := tw.Flush(); err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}
