package internal

import (
	"testing"

	"gioui.org/font"
)

func TestMarkdownBlocksStylesHeadingsAndInlineEmphasis(t *testing.T) {
	cfg := defaultConfig()
	blocks := markdownBlocks("# Title\n\nBody with **bold** and *italic*.", cfg, "")

	if got, want := blocks[0].Text, "Title"; got != want {
		t.Fatalf("heading text = %q, want %q", got, want)
	}
	if blocks[0].Font.Weight != font.Bold {
		t.Fatalf("heading weight = %v, want bold", blocks[0].Font.Weight)
	}
	if blocks[0].Size <= blocks[2].Size {
		t.Fatalf("heading size = %v, body size = %v; heading should be larger", blocks[0].Size, blocks[2].Size)
	}

	body := blocks[2]
	if len(body.Segments) != 5 {
		t.Fatalf("body segment count = %d, want 5", len(body.Segments))
	}
	if body.Segments[1].Text != "bold" || body.Segments[1].Font.Weight != font.Bold {
		t.Fatalf("bold segment = %#v, want bold text", body.Segments[1])
	}
	if body.Segments[3].Text != "italic" || body.Segments[3].Font.Style != font.Italic {
		t.Fatalf("italic segment = %#v, want italic text", body.Segments[3])
	}
}

func TestMarkdownBlocksResolvesLocalImages(t *testing.T) {
	cfg := defaultConfig()
	blocks := markdownBlocks("![diagram](images/flow.png){w=320}", cfg, "/vault/note-dir")

	if len(blocks) != 1 {
		t.Fatalf("block count = %d, want 1", len(blocks))
	}
	if got, want := blocks[0].ImagePath, "/vault/note-dir/images/flow.png"; got != want {
		t.Fatalf("image path = %q, want %q", got, want)
	}
	if got, want := blocks[0].ImageWidth, 320; got != want {
		t.Fatalf("image width = %d, want %d", got, want)
	}
}

func TestMarkdownBlocksTreatsBareImagePathAsImage(t *testing.T) {
	cfg := defaultConfig()
	blocks := markdownBlocks("/tmp/photo.png", cfg, "/vault/note-dir")

	if len(blocks) != 1 {
		t.Fatalf("block count = %d, want 1", len(blocks))
	}
	if got, want := blocks[0].ImagePath, "/tmp/photo.png"; got != want {
		t.Fatalf("image path = %q, want %q", got, want)
	}
}

func TestMarkdownBlocksNormalizesWindowsSeparators(t *testing.T) {
	cfg := defaultConfig()
	blocks := markdownBlocks("![diagram](images\\flow.png)", cfg, "/vault/note-dir")

	if len(blocks) != 1 {
		t.Fatalf("block count = %d, want 1", len(blocks))
	}
	if got, want := blocks[0].ImagePath, "/vault/note-dir/images/flow.png"; got != want {
		t.Fatalf("image path = %q, want %q", got, want)
	}
}

func TestMarkdownBlocksParsesTables(t *testing.T) {
	cfg := defaultConfig()
	blocks := markdownBlocks("| A | B |\n| --- | --- |\n| 1 | 2 |", cfg, "")

	if len(blocks) != 1 {
		t.Fatalf("block count = %d, want 1", len(blocks))
	}
	if got, want := blocks[0].Table[0][0], "A"; got != want {
		t.Fatalf("header cell = %q, want %q", got, want)
	}
	if got, want := blocks[0].Table[1][1], "2"; got != want {
		t.Fatalf("body cell = %q, want %q", got, want)
	}
}

func TestLooksLikeImagePath(t *testing.T) {
	for _, value := range []string{"path/to/image", "image.png", "/tmp/photo.jpeg"} {
		if !looksLikeImagePath(value) {
			t.Fatalf("%q should be treated as an image path", value)
		}
	}
	if looksLikeImagePath("plain selected text") {
		t.Fatal("plain selected text should not be treated as an image path")
	}
}

func TestFindReplaceHelpers(t *testing.T) {
	matches := findMatchRanges("One one ONE", "one")
	if len(matches) != 3 {
		t.Fatalf("match count = %d, want 3", len(matches))
	}
	got, count := replaceAllCaseInsensitive("One one ONE", "one", "two")
	if got != "two two two" || count != 3 {
		t.Fatalf("replace = %q/%d, want two two two/3", got, count)
	}
}
