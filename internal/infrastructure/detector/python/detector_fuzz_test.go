package python

import (
	"os"
	"testing"
)

func FuzzPythonDetector(f *testing.F) {
	seeds := []string{
		"import os\nimport sys\n",
		"from typing import List, Optional\n",
		"from .models import User\nfrom ..utils import helper\n",
		"import os\n\ndef test():\n    pass\n",
		"import numpy as np\nimport pandas as pd\n",
		"from collections import defaultdict, Counter\n",
		"import json\nfrom pathlib import Path\nimport sys\nfrom datetime import datetime\n",
		"import __future__\n",
		"from __future__ import annotations\n",
		"import os, sys, json\nfrom pathlib import Path as P\n",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, content string) {
	tmpDir := t.TempDir()
	filePath := tmpDir + "/test.py"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return
	}
	d := New()
	// Should never panic, even with malformed Python
	deps, err := d.parseFile(filePath, tmpDir, nil)
	if err != nil {
		return // Parse errors expected for malformed input
	}
	_ = deps
	})
}
