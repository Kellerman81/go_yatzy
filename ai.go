package main

// AIDecide determines which dice to keep given the current dice values and
// which categories are already scored. Returns the keep-flags.
func AIDecide(dice [NumDice]int, scored [NumCategories]bool) [NumDice]bool {
	bestCat := AIChooseTargetCategory(dice, scored)
	if bestCat < 0 {
		var keep [NumDice]bool
		for i := range keep {
			keep[i] = true
		}
		return keep
	}
	return keepDiceForCategory(bestCat, dice)
}

// AIChooseCategory picks the best available category to score right now.
func AIChooseCategory(dice [NumDice]int, scored [NumCategories]bool) int {
	bestCat, bestScore := -1, -2
	for cat := 0; cat < NumCategories; cat++ {
		if scored[cat] {
			continue
		}
		score := CalculateScore(cat, dice)
		adjusted := score
		if score > 0 {
			adjusted += 10
		}
		if adjusted > bestScore {
			bestScore = adjusted
			bestCat = cat
		}
	}
	return bestCat
}

// AIChooseTargetCategory picks the best category to aim for (not necessarily to score now).
func AIChooseTargetCategory(dice [NumDice]int, scored [NumCategories]bool) int {
	bestCat, bestPotential := -1, -1
	for cat := 0; cat < NumCategories; cat++ {
		if scored[cat] {
			continue
		}
		p := potential(cat, dice)
		if p > bestPotential {
			bestPotential = p
			bestCat = cat
		}
	}
	return bestCat
}

func potential(cat int, dice [NumDice]int) int {
	current := CalculateScore(cat, dice)
	counts := countDice(dice)

	switch cat {
	case CatOnes:
		return current + counts[1]*2
	case CatTwos:
		return current + counts[2]*2
	case CatThrees:
		return current + counts[3]*2
	case CatFours:
		return current + counts[4]*2
	case CatFives:
		return current + counts[5]*2
	case CatSixes:
		return current + counts[6]*2
	case CatThreeOfKind, CatFourOfKind:
		maxCount := 0
		for _, c := range counts {
			if c > maxCount {
				maxCount = c
			}
		}
		return current + maxCount*4
	case CatFullHouse:
		if current > 0 {
			return 55
		}
		hasThree, hasTwo := false, false
		for _, c := range counts {
			if c >= 3 {
				hasThree = true
			} else if c == 2 {
				hasTwo = true
			}
		}
		if hasThree || hasTwo {
			return 20
		}
		return 5
	case CatSmallStraight:
		if current > 0 {
			return 50
		}
		return seqLen(counts) * 7
	case CatLargeStraight:
		if current > 0 {
			return 60
		}
		return seqLen(counts) * 9
	case CatKniffel:
		maxCount := 0
		for _, c := range counts {
			if c > maxCount {
				maxCount = c
			}
		}
		if maxCount == 5 {
			return 80
		}
		return maxCount * 16
	case CatChance:
		s := 0
		for _, v := range dice {
			s += v
		}
		return s + 5
	}
	return 0
}

func seqLen(counts [7]int) int {
	maxLen, curLen := 0, 0
	for i := 1; i <= 6; i++ {
		if counts[i] > 0 {
			curLen++
			if curLen > maxLen {
				maxLen = curLen
			}
		} else {
			curLen = 0
		}
	}
	return maxLen
}

func keepDiceForCategory(cat int, dice [NumDice]int) [NumDice]bool {
	var keep [NumDice]bool
	counts := countDice(dice)

	switch {
	case cat <= CatSixes:
		target := cat + 1
		for i, v := range dice {
			if v == target {
				keep[i] = true
			}
		}

	case cat == CatThreeOfKind || cat == CatFourOfKind || cat == CatKniffel:
		maxVal, maxCount := 0, 0
		for v, c := range counts {
			if c > maxCount {
				maxCount = c
				maxVal = v
			}
		}
		for i, v := range dice {
			if v == maxVal {
				keep[i] = true
			}
		}

	case cat == CatFullHouse:
		// Keep groups of 2 or 3
		for i, v := range dice {
			if counts[v] >= 2 {
				keep[i] = true
			}
		}

	case cat == CatSmallStraight || cat == CatLargeStraight:
		inSeq := bestSeqValues(counts)
		used := make(map[int]bool)
		for i, v := range dice {
			if inSeq[v] && !used[v] {
				keep[i] = true
				used[v] = true
			}
		}

	case cat == CatChance:
		for i, v := range dice {
			if v >= 4 {
				keep[i] = true
			}
		}
	}
	return keep
}

func bestSeqValues(counts [7]int) map[int]bool {
	best := []int{}
	cur := []int{}
	for i := 1; i <= 6; i++ {
		if counts[i] > 0 {
			cur = append(cur, i)
		} else {
			if len(cur) > len(best) {
				best = make([]int, len(cur))
				copy(best, cur)
			}
			cur = cur[:0]
		}
	}
	if len(cur) > len(best) {
		best = cur
	}
	result := make(map[int]bool)
	for _, v := range best {
		result[v] = true
	}
	return result
}
