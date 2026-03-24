//go:build !solution

package main

import (
	"fmt"
	"strconv"
	"strings"
)

type wordFunc func(stack []int) ([]int, error)

type Evaluator struct {
	words map[string]wordFunc
	stack []int
}

// NewEvaluator creates evaluator.
func NewEvaluator() *Evaluator {
	e := &Evaluator{words: make(map[string]wordFunc)}

	e.words["+"] = func(stack []int) ([]int, error) {
		if len(stack) < 2 {
			return nil, fmt.Errorf("not enough arguments for +")
		}
		a, b := stack[len(stack)-2], stack[len(stack)-1]
		return append(stack[:len(stack)-2], a+b), nil
	}

	e.words["-"] = func(stack []int) ([]int, error) {
		if len(stack) < 2 {
			return nil, fmt.Errorf("not enough arguments for -")
		}
		a, b := stack[len(stack)-2], stack[len(stack)-1]
		return append(stack[:len(stack)-2], a-b), nil
	}

	e.words["*"] = func(stack []int) ([]int, error) {
		if len(stack) < 2 {
			return nil, fmt.Errorf("not enough arguments for *")
		}
		a, b := stack[len(stack)-2], stack[len(stack)-1]
		return append(stack[:len(stack)-2], a*b), nil
	}

	e.words["/"] = func(stack []int) ([]int, error) {
		if len(stack) < 2 {
			return nil, fmt.Errorf("not enough arguments for /")
		}
		a, b := stack[len(stack)-2], stack[len(stack)-1]
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return append(stack[:len(stack)-2], a/b), nil
	}

	e.words["dup"] = func(stack []int) ([]int, error) {
		if len(stack) < 1 {
			return nil, fmt.Errorf("not enough arguments for dup")
		}
		return append(stack, stack[len(stack)-1]), nil
	}

	e.words["drop"] = func(stack []int) ([]int, error) {
		if len(stack) < 1 {
			return nil, fmt.Errorf("not enough arguments for drop")
		}
		return stack[:len(stack)-1], nil
	}

	e.words["swap"] = func(stack []int) ([]int, error) {
		if len(stack) < 2 {
			return nil, fmt.Errorf("not enough arguments for swap")
		}
		stack[len(stack)-1], stack[len(stack)-2] = stack[len(stack)-2], stack[len(stack)-1]
		return stack, nil
	}

	e.words["over"] = func(stack []int) ([]int, error) {
		if len(stack) < 2 {
			return nil, fmt.Errorf("not enough arguments for over")
		}
		return append(stack, stack[len(stack)-2]), nil
	}

	return e
}

func (e *Evaluator) Process(row string) ([]int, error) {
	tokens := strings.Fields(row)

	if len(tokens) >= 1 && tokens[0] == ":" {
		return e.defineWord(tokens)
	}

	return e.evalTokens(tokens, e.stack)
}

func (e *Evaluator) defineWord(tokens []string) ([]int, error) {
	if len(tokens) < 4 || tokens[len(tokens)-1] != ";" {
		return nil, fmt.Errorf("invalid word definition")
	}

	word_name := strings.ToLower(tokens[1])
	if _, err := strconv.Atoi(word_name); err == nil {
		return nil, fmt.Errorf("cannot redefine numbers")
	}

	body_tokens := make([]string, len(tokens[2:len(tokens)-1]))
	copy(body_tokens, tokens[2:len(tokens)-1])

	captured_funcs := make([]wordFunc, len(body_tokens))
	for i, t := range body_tokens {
		lower_t := strings.ToLower(t)
		if fn, ok := e.words[lower_t]; ok {
			captured_funcs[i] = fn
		}
	}

	e.words[word_name] = func(stack []int) ([]int, error) {
		var err error
		for i, t := range body_tokens {
			if captured_funcs[i] != nil {
				stack, err = captured_funcs[i](stack)
				if err != nil {
					return nil, err
				}
				continue
			}
			num, parse_err := strconv.Atoi(t)
			if parse_err != nil {
				return nil, fmt.Errorf("unknown word: %s", t)
			}
			stack = append(stack, num)
		}
		return stack, nil
	}

	return e.stack, nil
}

func (e *Evaluator) evalTokens(tokens []string, stack []int) ([]int, error) {
	var err error
	for _, t := range tokens {
		lowerT := strings.ToLower(t)
		if fn, ok := e.words[lowerT]; ok {
			stack, err = fn(stack)
			if err != nil {
				return nil, err
			}
			continue
		}
		num, parseErr := strconv.Atoi(t)
		if parseErr != nil {
			return nil, fmt.Errorf("unknown word: %s", t)
		}
		stack = append(stack, num)
	}
	e.stack = stack
	return stack, nil
}
