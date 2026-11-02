package vault

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// Wikilinks returns the slugs that text links to with [[slug]], [[slug|alias]], [[slug#heading]]
// or an embed, once each in order of first appearance. A link to a path counts as its base name.
// Links inside code fences and inline code spans are ignored; a code span must open and close on
// one line.
func Wikilinks(text string) []string {
	var slugs []string
	seen := map[string]bool{}
	fence := fenceTracker{}
	for line := range strings.SplitSeq(text, "\n") {
		if fence.inCode(line) {
			continue
		}
		for _, match := range wikilinkPattern.FindAllStringSubmatch(withoutCodeSpans(line), -1) {
			slug := linkTarget(match[1])
			if slug != "" && !seen[slug] {
				seen[slug] = true
				slugs = append(slugs, slug)
			}
		}
	}
	return slugs
}

// linkTarget reduces the inside of [[...]] to a slug. A table cell escapes the alias pipe as \|.
func linkTarget(inner string) string {
	target, _, _ := strings.Cut(inner, "|")
	target, _, _ = strings.Cut(target, "#")
	target = strings.TrimSpace(strings.TrimSuffix(target, `\`))
	if target == "" {
		return ""
	}
	return strings.TrimSuffix(path.Base(target), noteExtension)
}

// withoutCodeSpans blanks every inline code span: a backtick run up to the next run of the same
// length. An unmatched run is literal text.
func withoutCodeSpans(line string) string {
	runs := backtickRun.FindAllStringIndex(line, -1)
	if len(runs) < 2 {
		return line
	}
	blanked := []byte(line)
	for open := 0; open < len(runs); open++ {
		length := runs[open][1] - runs[open][0]
		for close := open + 1; close < len(runs); close++ {
			if runs[close][1]-runs[close][0] != length {
				continue
			}
			for index := runs[open][0]; index < runs[close][1]; index++ {
				blanked[index] = ' '
			}
			open = close
			break
		}
	}
	return string(blanked)
}

// Backlinks returns the paths of the notes under root that link to slug, in lexical order. Hidden
// directories such as .obsidian are skipped.
func Backlinks(root, slug string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if current != root && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(current) != noteExtension {
			return nil
		}
		text, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		if slices.Contains(Wikilinks(string(text)), slug) {
			paths = append(paths, current)
		}
		return nil
	})
	return paths, err
}
