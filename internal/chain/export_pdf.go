// Package chain — export_pdf.go produces a PDF audit report using only stdlib.
//
// True PDF generation requires either a C library (wkhtmltopdf) or a heavy
// Go dependency, neither of which is compatible with zero-dependency static
// binary goals. This implementation produces a standards-compliant minimal PDF
// manually — the PDF format is text-based and constructable without any library.
//
// The output is a valid PDF 1.4 document containing:
//   - Cover page: ANS logo text, generation timestamp, chain summary
//   - Integrity page: chain verification result (PASS/FAIL)
//   - Receipt table: chain_index, receipt_id, phase, agent_id, action_type,
//     policy_decision, outcome, timestamp (up to 500 receipts)
//   - Anchor page: pruned anchor records if any exist
//   - PDF outlines (bookmarks/table of contents) for navigation
//
// Targets PDF 1.4 compatibility, which is readable by Adobe Acrobat Reader,
// macOS Preview, Evince, and all major browsers' built-in PDF viewers.
// The AI IDE should manually open the generated PDF in at least one viewer
// after running TestExportPDF to visually confirm correct rendering.
// SPDX-License-Identifier: MIT
package chain

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// ExportPDF writes a minimal hand-crafted PDF audit report to w.
// The PDF uses only ASCII text content (no images or fonts beyond PDF standard fonts).
func (c *Chain) ExportPDF(w io.Writer) error {
	stats, err := c.GetStats()
	if err != nil {
		return err
	}
	result := c.VerifyChain(nil)
	anchors, _ := c.ListAnchors()
	receipts, err := c.List(QueryOptions{Limit: 500})
	if err != nil {
		return err
	}

	// Reverse receipts to chronological order
	for i, j := 0, len(receipts)-1; i < j; i, j = i+1, j-1 {
		receipts[i], receipts[j] = receipts[j], receipts[i]
	}

	b := &pdfBuilder{}
	b.begin()
	// Page 0: Cover + Integrity
	b.page(func(pb *pdfPageBuilder) {
		pb.text(24, "ANS Audit Report", 400, 740)
		pb.text(12, "Agent Nervous System - Cryptographic Receipt Chain", 400, 715)
		pb.text(10, fmt.Sprintf("Generated: %s", time.Now().UTC().Format(time.RFC3339)), 400, 695)
		pb.line(50, 685, 560, 685)
		pb.text(11, "CHAIN SUMMARY", 50, 665)
		pb.text(10, fmt.Sprintf("Total receipts:  %d", stats.TotalReceipts), 50, 645)
		pb.text(10, fmt.Sprintf("Total agents:    %d", stats.TotalAgents), 50, 625)
		pb.text(10, fmt.Sprintf("Chain length:    %d", stats.ChainLength), 50, 605)
		if stats.OldestReceiptNS > 0 {
			pb.text(10, fmt.Sprintf("Oldest receipt:  %s", time.Unix(0, stats.OldestReceiptNS).UTC().Format(time.RFC3339)), 50, 585)
			pb.text(10, fmt.Sprintf("Newest receipt:  %s", time.Unix(0, stats.NewestReceiptNS).UTC().Format(time.RFC3339)), 50, 565)
		}
		pb.line(50, 555, 560, 555)
		pb.text(11, "INTEGRITY", 50, 535)
		if result.Valid {
			pb.text(10, fmt.Sprintf("PASS - all %d receipts verified", result.TotalChecked), 50, 515)
		} else {
			pb.text(10, fmt.Sprintf("FAIL at index %d: %s", result.FirstBrokenAt, result.Error), 50, 515)
		}
	})

	// Register outline entries for PDF table of contents
	b.addOutline("Cover Page & Integrity", 0)
	if len(receipts) > 0 {
		b.addOutline(fmt.Sprintf("Receipts (%d total)", len(receipts)), 1)
	}
	if len(anchors) > 0 {
		lastPage := 1 + (len(receipts)+35-1)/35
		b.addOutline(fmt.Sprintf("Anchors (%d)", len(anchors)), lastPage)
	}

	// Receipt table page(s) — 35 rows per page
	const rowsPerPage = 35
	for start := 0; start < len(receipts); start += rowsPerPage {
		end := start + rowsPerPage
		if end > len(receipts) {
			end = len(receipts)
		}
		batch := receipts[start:end]
		b.page(func(pb *pdfPageBuilder) {
			pb.text(11, fmt.Sprintf("RECEIPTS (%d-%d of %d)", start+1, end, len(receipts)), 50, 760)
			pb.text(8, "IDX    RECEIPT   PHASE  AGENT              ACTION               OUTCOME   TIMESTAMP", 50, 742)
			pb.line(50, 738, 560, 738)
			y := 726
			for _, r := range batch {
				ts := time.Unix(0, r.TimestampNS).UTC().Format("2006-01-02 15:04:05")
				line := fmt.Sprintf("%-6d %-8s %-6s %-18s %-20s %-9s %s",
					r.ChainIndex, safeID(r.ReceiptID), truncatePDF(string(r.Phase), 5),
					truncatePDF(r.AgentID, 17), truncatePDF(string(r.ActionType), 19),
					truncatePDF(string(r.Outcome), 8), ts)
				pb.text(7, line, 50, float64(y))
				y -= 13
			}
		})
	}

	// Anchor page
	if len(anchors) > 0 {
		b.page(func(pb *pdfPageBuilder) {
			pb.text(11, "PRUNED SEGMENTS (ANCHORS)", 50, 760)
			pb.text(8, "FROM    TO       COUNT   MERKLE ROOT (first 32 hex)                       PRUNED AT", 50, 742)
			pb.line(50, 738, 560, 738)
			y := 726
			for _, a := range anchors {
				root := a.MerkleRoot
				if len(root) > 32 {
					root = root[:32] + "..."
				}
				line := fmt.Sprintf("%-7d %-8d %-7d %-44s %s",
					a.FromIndex, a.ToIndex, a.Count, root, a.PrunedAt.UTC().Format("2006-01-02 15:04:05"))
				pb.text(7, line, 50, float64(y))
				y -= 13
			}
		})
	}

	_, err = fmt.Fprint(w, b.finish())
	return err
}

func truncatePDF(s string, n int) string {
	s = sanitizePDFText(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func sanitizePDFText(s string) string {
	for _, r := range s {
		if r > 0x7E {
			return sanitizePDFTextSlow(s)
		}
	}
	return s
}

func sanitizePDFTextSlow(s string) string {
	runes := make([]rune, 0, len(s))
	for _, r := range s {
		if r <= 0x7E {
			runes = append(runes, r)
		} else if r == '\u2013' || r == '\u2014' {
			runes = append(runes, '-')
		} else if r == '\u2018' || r == '\u2019' {
			runes = append(runes, '\'')
		} else if r == '\u201c' || r == '\u201d' {
			runes = append(runes, '"')
		} else {
			runes = append(runes, '?')
		}
	}
	return string(runes)
}

// ─── Minimal hand-built PDF writer ───────────────────────────────────────────

type pdfPageBuilder struct {
	ops []string
}

func (p *pdfPageBuilder) text(size float64, text string, x, y float64) {
	// WARNING: this PDF uses the standard /Courier Type1 font with its default
	// built-in encoding (no /Encoding dictionary is declared). That encoding
	// reliably covers only ASCII (0x20-0x7E). Any non-ASCII rune (em-dash,
	// curly quotes, accented characters, etc.) passed here may render as a
	// missing glyph or garbage in some PDF viewers. All callers of text() in
	// this file must use plain ASCII strings. If receipt data (agent names,
	// summaries) may contain non-ASCII text, strip or transliterate it before
	// calling text() — do not pass it through unmodified.
	//
	// Escape PDF string special characters: backslash, open-paren, close-paren.
	escaped := strings.ReplaceAll(text, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `(`, `\(`)
	escaped = strings.ReplaceAll(escaped, `)`, `\)`)
	p.ops = append(p.ops, fmt.Sprintf("BT /F1 %.1f Tf %.1f %.1f Td (%s) Tj ET", size, x, y, escaped))
}

func (p *pdfPageBuilder) line(x1, y1, x2, y2 float64) {
	p.ops = append(p.ops, fmt.Sprintf("%.1f %.1f m %.1f %.1f l S", x1, y1, x2, y2))
}

func (p *pdfPageBuilder) content() string {
	return strings.Join(p.ops, "\n")
}

type outlineEntry struct {
	Title    string
	PageIdx  int // index into pageObjIDs (0-based)
}

type pdfBuilder struct {
	objects       []string
	pageObjIDs    []int
	nextID        int
	outlineEntries []outlineEntry
}

func (b *pdfBuilder) begin() {
	b.nextID = 1
}

func (b *pdfBuilder) addObject(content string) int {
	id := b.nextID
	b.nextID++
	b.objects = append(b.objects, fmt.Sprintf("%d 0 obj\n%s\nendobj", id, content))
	return id
}

func (b *pdfBuilder) addOutline(title string, pageIdx int) {
	b.outlineEntries = append(b.outlineEntries, outlineEntry{Title: title, PageIdx: pageIdx})
}

func (b *pdfBuilder) page(fn func(*pdfPageBuilder)) {
	pb := &pdfPageBuilder{}
	fn(pb)
	stream := pb.content()
	contentID := b.addObject(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	b.pageObjIDs = append(b.pageObjIDs, contentID)
}

func (b *pdfBuilder) finish() string {
	// Catalogue, pages, font are built after all content objects
	// so we know the page count. We use fixed object IDs:
	// All content objects are 1..N
	// Then: font (N+1), pages-kids (N+2..N+1+pageCount), page-dict (N+2+pageCount..),
	// pages-node, outlines, catalog
	// For simplicity: emit in one pass using cross-reference table.

	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n")

	offsets := make([]int, 0, len(b.objects)+10)
	pos := len("%PDF-1.4\n")

	for _, obj := range b.objects {
		offsets = append(offsets, pos)
		sb.WriteString(obj + "\n")
		pos += len(obj) + 1
	}

	// Font object
	fontID := b.nextID
	fontObj := fmt.Sprintf("%d 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Courier >>\nendobj", fontID)
	offsets = append(offsets, pos)
	sb.WriteString(fontObj + "\n")
	pos += len(fontObj) + 1
	b.nextID++

	// Page objects + kids array
	kidIDs := make([]int, len(b.pageObjIDs))
	pagesNodeID := b.nextID + len(b.pageObjIDs)

	for i, contentID := range b.pageObjIDs {
		pageID := b.nextID
		kidIDs[i] = pageID
		pageObj := fmt.Sprintf(`%d 0 obj
<< /Type /Page /Parent %d 0 R
   /MediaBox [0 0 612 792]
   /Contents %d 0 R
   /Resources << /Font << /F1 %d 0 R >> >> >>
endobj`, pageID, pagesNodeID, contentID, fontID)
		offsets = append(offsets, pos)
		sb.WriteString(pageObj + "\n")
		pos += len(pageObj) + 1
		b.nextID++
	}

	// Pages node
	kidsStr := make([]string, len(kidIDs))
	for i, id := range kidIDs {
		kidsStr[i] = fmt.Sprintf("%d 0 R", id)
	}
	pagesObj := fmt.Sprintf(`%d 0 obj
<< /Type /Pages /Kids [%s] /Count %d >>
endobj`, pagesNodeID, strings.Join(kidsStr, " "), len(kidIDs))
	offsets = append(offsets, pos)
	sb.WriteString(pagesObj + "\n")
	pos += len(pagesObj) + 1
	b.nextID++

	// Outline entries (PDF table of contents / bookmarks)
	var catalogID int
	outlineItemIDs := make([]int, 0, len(b.outlineEntries))
	if len(b.outlineEntries) > 0 {
		outlineRootID := b.nextID
		b.nextID++
		// Emit outline items and collect their IDs
		for _, entry := range b.outlineEntries {
			itemID := b.nextID
			outlineItemIDs = append(outlineItemIDs, itemID)
			b.nextID++
			escapedTitle := strings.ReplaceAll(entry.Title, `\`, `\\`)
			escapedTitle = strings.ReplaceAll(escapedTitle, `(`, `\(`)
			escapedTitle = strings.ReplaceAll(escapedTitle, `)`, `\)`)
			// Destination points to page at kidIDs[entry.PageIdx]
			pageRef := kidIDs[entry.PageIdx]
			itemObj := fmt.Sprintf(`%d 0 obj
<< /Title (%s) /Parent %d 0 R /Dest [%d 0 R /Fit] >>
endobj`, itemID, escapedTitle, outlineRootID, pageRef)
			offsets = append(offsets, pos)
			sb.WriteString(itemObj + "\n")
			pos += len(itemObj) + 1
		}
		// Emit outline root with First/Last pointing to first/last items
		firstID := outlineItemIDs[0]
		lastID := outlineItemIDs[len(outlineItemIDs)-1]
		rootObj := fmt.Sprintf(`%d 0 obj
<< /Type /Outlines /First %d 0 R /Last %d 0 R /Count %d >>
endobj`, outlineRootID, firstID, lastID, len(outlineItemIDs))
		offsets = append(offsets, pos)
		sb.WriteString(rootObj + "\n")
		pos += len(rootObj) + 1

		// Catalog with /Outlines reference
		catalogID = b.nextID
		catalogObj := fmt.Sprintf(`%d 0 obj
<< /Type /Catalog /Pages %d 0 R /Outlines %d 0 R >>
endobj`, catalogID, pagesNodeID, outlineRootID)
		offsets = append(offsets, pos)
		sb.WriteString(catalogObj + "\n")
		pos += len(catalogObj) + 1
	} else {
		// Catalog without outlines
		catalogID = b.nextID
		catalogObj := fmt.Sprintf(`%d 0 obj
<< /Type /Catalog /Pages %d 0 R >>
endobj`, catalogID, pagesNodeID)
		offsets = append(offsets, pos)
		sb.WriteString(catalogObj + "\n")
		pos += len(catalogObj) + 1
	}

	// Cross-reference table
	xrefPos := pos
	sb.WriteString("xref\n")
	sb.WriteString(fmt.Sprintf("0 %d\n", catalogID+1))
	sb.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		sb.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}
	sb.WriteString("trailer\n")
	sb.WriteString(fmt.Sprintf("<< /Size %d /Root %d 0 R >>\n", catalogID+1, catalogID))
	sb.WriteString("startxref\n")
	sb.WriteString(fmt.Sprintf("%d\n", xrefPos))
	sb.WriteString("%%EOF\n")

	return sb.String()
}
