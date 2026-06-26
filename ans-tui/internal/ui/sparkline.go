package ui

import "strings"

var sparkChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func RenderSparkline(values []float64, width int) string {
	if len(values) == 0 {
		return strings.Repeat(string(sparkChars[0]), width)
	}
	sampled := resample(values, width)
	m := maxFloat(sampled)
	if m == 0 {
		m = 1
	}
	var sb strings.Builder
	for _, v := range sampled {
		idx := int((v / m) * float64(len(sparkChars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkChars) {
			idx = len(sparkChars) - 1
		}
		sb.WriteRune(sparkChars[idx])
	}
	return sb.String()
}

func resample(in []float64, target int) []float64 {
	if len(in) == target {
		out := make([]float64, target)
		copy(out, in)
		return out
	}
	out := make([]float64, target)
	for i := range out {
		srcIdx := float64(i) * float64(len(in)) / float64(target)
		lo := int(srcIdx)
		if lo >= len(in)-1 {
			lo = len(in) - 2
		}
		if lo < 0 {
			lo = 0
		}
		hi := lo + 1
		if hi >= len(in) {
			hi = len(in) - 1
		}
		frac := srcIdx - float64(lo)
		out[i] = in[lo]*(1-frac) + in[hi]*frac
	}
	return out
}

func maxFloat(s []float64) float64 {
	m := s[0]
	for _, v := range s[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
