---
layout: "localfile"
page_title: "Localfile: localfile_file"
sidebar_current: "docs-localfile-resource-file"
description: |-
  Generates a local file from content.
---

# localfile\_file

Generates a local file from content.

## Example Usage

```
data "localfile_file" "foo" {
    content     = "foo!"
    destination = "${path.module}/foo.bar"
}
```

## Argument Reference

The following arguments are supported:

* `content` - (required) The content of file to create.

* `destination` - (required) The path of the file to create.

NOTE: Any required folders are created.
