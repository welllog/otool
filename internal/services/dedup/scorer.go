package dedup

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// EvaluateAndScoreGroup evaluates all photos in a group, scores them, and assigns BestPickID.
// It also applies the default delete selection (all non-best photos).
func EvaluateAndScoreGroup(group *DuplicateGroup) {
	evaluateAndScoreGroup(group, false)
}

// EvaluateAndScoreGroupPreserveSelection re-scores a group without resetting the
// user's Selected flags. Use this after a partial cleanup so photos the user
// explicitly chose to keep do not become selected for deletion again.
func EvaluateAndScoreGroupPreserveSelection(group *DuplicateGroup) {
	evaluateAndScoreGroup(group, true)
}

func evaluateAndScoreGroup(group *DuplicateGroup, preserveSelection bool) {
	if len(group.Photos) == 0 {
		return
	}

	// 1. Find group extrema
	var maxPixels int64
	var maxBlur float64
	var maxSize int64

	for _, p := range group.Photos {
		pixels := int64(p.Width) * int64(p.Height)
		if pixels > maxPixels {
			maxPixels = pixels
		}
		if p.BlurScore > maxBlur {
			maxBlur = p.BlurScore
		}
		if p.Size > maxSize {
			maxSize = p.Size
		}
	}

	if maxPixels <= 0 {
		maxPixels = 1
	}
	if maxBlur <= 0 {
		maxBlur = 1
	}
	if maxSize <= 0 {
		maxSize = 1
	}

	// 2. Score each photo
	for _, p := range group.Photos {
		var score float64
		var reasons []string

		// (a) Resolution score (max 40 pts)
		pixels := int64(p.Width) * int64(p.Height)
		resRatio := float64(pixels) / float64(maxPixels)
		score += resRatio * 40.0
		if resRatio >= 0.99 && pixels > 1920*1080 {
			reasons = append(reasons, fmt.Sprintf("最高分辨率 (%dx%d)", p.Width, p.Height))
		} else if resRatio < 0.6 {
			reasons = append(reasons, "分辨率较低")
		}

		// (b) Sharpness score (max 25 pts)
		blurRatio := p.BlurScore / maxBlur
		score += blurRatio * 25.0
		if maxBlur > 100 {
			if blurRatio >= 0.85 {
				reasons = append(reasons, "画面锐利清晰")
			} else if blurRatio < 0.5 {
				reasons = append(reasons, "画面偏模糊/对焦失准")
			}
		}

		// (c) EXIF metadata completeness (max 20 pts)
		if p.HasEXIF {
			if p.CameraModel != "" {
				score += 10.0
				reasons = append(reasons, fmt.Sprintf("原片 EXIF (%s)", p.CameraModel))
			} else {
				score += 5.0
			}
			if !p.DateTimeOrig.IsZero() {
				score += 10.0
			}
		} else {
			reasons = append(reasons, "缺少 EXIF 元数据")
		}

		// (d) File size / compression quality (max 15 pts)
		sizeRatio := float64(p.Size) / float64(maxSize)
		score += sizeRatio * 15.0

		// (e) Filename penalties
		lowerName := strings.ToLower(p.Name)
		if strings.Contains(lowerName, "_copy") ||
			strings.Contains(lowerName, " (1)") ||
			strings.Contains(lowerName, "(1)") ||
			strings.Contains(lowerName, "wechat_") ||
			strings.Contains(lowerName, "wx_camera") ||
			strings.Contains(lowerName, "thumb") {
			score -= 15.0
			reasons = append(reasons, "副本/转发特征命名")
		}

		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}

		p.Score = math.Round(score*10) / 10
		p.ScoreReasons = reasons
	}

	// 3. Sort photos by Score descending
	sort.Slice(group.Photos, func(i, j int) bool {
		if group.Photos[i].Score != group.Photos[j].Score {
			return group.Photos[i].Score > group.Photos[j].Score
		}
		// Tie-break: larger size
		return group.Photos[i].Size > group.Photos[j].Size
	})

	// 4. Mark BestPick and default selections
	bestPick := group.Photos[0]
	if preserveSelection {
		// After a partial cleanup the scores may rank a photo the user
		// explicitly marked for deletion above every keeper. Crowning that
		// photo "recommended to keep" while its cleanup checkbox stays
		// checked is contradictory, so crown the best-scoring keeper
		// instead. The deletion flow always leaves at least one unselected
		// photo per group; the Photos[0] fallback above only guards other
		// callers.
		for _, p := range group.Photos {
			if !p.Selected {
				bestPick = p
				break
			}
		}
	}
	group.BestPickID = bestPick.ID

	var wastedBytes int64
	if preserveSelection {
		// Keep the user's checked state intact after a partial cleanup.
		for _, p := range group.Photos {
			if p.Selected {
				wastedBytes += p.Size
			}
		}
	} else {
		// BestPick reasons highlight
		bestPick.Selected = false // Unchecked for deletion (KEEP)

		for i := 1; i < len(group.Photos); i++ {
			group.Photos[i].Selected = true // Checked for deletion by default
			wastedBytes += group.Photos[i].Size
		}
	}
	group.TotalWastedBytes = wastedBytes

	// Calculate average similarity percent
	if group.Mode == ModeExact {
		group.SimilarityPercent = 100
	} else {
		maxDistInGroup := 0
		for _, p := range group.Photos {
			if p.ID != bestPick.ID {
				d, _ := ArePhotosVisuallySimilar(bestPick, p, 64)
				if d > maxDistInGroup {
					maxDistInGroup = d
				}
			}
		}
		pct := 100 - (maxDistInGroup * 100 / 64)
		if pct > 100 {
			pct = 100
		}
		if pct < 70 {
			pct = 70
		}
		group.SimilarityPercent = pct
	}
}
