// Package lexer makes it easy to build lexical analyzers.
//
// Lexical analysis is a common problem in computer science; how can a computer program
// interpret a sequence of characters in a meaningful way? The canonical example, a
// compiler, must be able to transform source code written in a programming language into
// its target language (i.e. object code).
//
// To accomplish this task a compiler will often employ a lexical analyzer to convert a
// sequence of characters (e.g. source code) into a sequence of meaningful groups of
// characters, or tokens. For example consider this simple Go expression:
//
//     x := y + 2.0
//
// Go's lexical analyzer would emit IDENT, DEFINE, IDENT, ADD, and FLOAT tokens. Notice
// that only the information that is relevant to the syntax of the expression is emitted
// as a token; later, semantic analysis, phases of the Go compiler are responsible for
// ensuring that the expression consists of valid Go source code.
//
// In 2011 Rob Pike presented a lexical analyzer written in Go at the Sydney Google
// Technology User Group (see http://youtu.be/HxaD_trXwRE). Pike's talk covered many
// topics but really converged on two powerful concepts:
//
//     1. Representing state and state changes as functions
//     2. Leveraging goroutines and channels to emit tokens
//
// A lexical analyzer can most often be implemented as a state machine. As a lexical
// analyzer traverses a sequence of characters it can be in one of any number of valid
// states. To transition between states the lexical analyzer must examine the next
// character in its sequence of characters and, based on the state it is currently in,
// determine which valid state to enter.
//
// Functions are a great way to represent states, and transitions between states, in a
// state machine. The lexical analyzer that is the subject of Pike's talk is inialized
// with a function representing its initial state. This function is then able to inspect
// the input and, based on what it encounters and its position in that input, can return
// the next state the lexical analyzer should transition to.
//
// The process of emitting tokens during lexical analysis can be greatly simplified by
// taking advantage of Go's concurrency primitives. Running the state machine that drives
// the lexical analyzer in a goroutine allows emitted tokens to be sent to a specific
// channel; decoupling the two responsibilities almost entirely.
//
// This package was created as an exercise in learning the Go programming language.
package lexer

import (
	"fmt"
	"sync"
	"unicode/utf8"
)

// Token, consisting of a type and value, represents the output of the lexer.
type Token struct {
	Type  TokenType
	Value interface{}
}

// TokenType represents the type of a given token.
type TokenType int

// TokenError represents a type of token that contains an error message as its value.
const TokenError TokenType = -1

// EOF represents the end of the input.
const EOF = rune(-1)

// RunePosition represents the position of a rune in the input.
type RunePosition int

// RuneWidth represents the width of a rune.
type RuneWidth int

// StateFunc is a function that represents the state of the lexer.
type StateFunc func(*Lexer) StateFunc

// RunePredicate is a function that returns true or false based on the specified rune.
type RunePredicate func(rune) bool

// Lexer contains the lexer's internal state.
type Lexer struct {
	Input            string
	CurrentPosition  RunePosition
	CurrentRuneWidth RuneWidth
	initialState     StateFunc
	startPosition    RunePosition
	currentToken     Token
	previousToken    Token
	tokenMutex       sync.Mutex
	tokens           chan Token
}

// NewLexer creates a lexer from the input and initial state.
func NewLexer(input string, initialState StateFunc) *Lexer {
	l := &Lexer{
		Input:        input,
		initialState: initialState,
		tokens:       make(chan Token, 1),
	}
	go func() {
		for s := l.initialState; s != nil; {
			s = s(l)
		}
	}()
	return l
}

// NextToken returns the next token emitted by the lexer.
func (l *Lexer) NextToken() Token {
	return <-l.tokens
}

// PreviousToken returns the most recently emitted token.
func (l *Lexer) PreviousToken() Token {
	l.tokenMutex.Lock()
	defer l.tokenMutex.Unlock()
	return l.previousToken
}

// Next returns the next rune from the input and moves the current position of the lexer
// ahead.
//
// If encountering the end of the input EOF will be returned.
func (l *Lexer) Next() rune {
	if int(l.CurrentPosition) >= len(l.Input) {
		l.CurrentRuneWidth = 0
		return EOF
	}
	r, w := utf8.DecodeRuneInString(l.Input[l.CurrentPosition:])
	l.CurrentRuneWidth = RuneWidth(w)
	l.CurrentPosition += RunePosition(l.CurrentRuneWidth)
	return r
}

// NextUpTo returns the rune last seen by the predicate and moves the current position of
// the lexer ahead.
//
// Returns EOF if the end of input is encountered before the predicate is satisfied.
func (l *Lexer) NextUpTo(predicate RunePredicate) rune {
	return l.consumeUpTo(predicate, l.Next)
}

// Peek returns the next rune from the input without moving the current position of the
// lexer ahead.
func (l *Lexer) Peek() rune {
	r := l.Next()
	l.Previous()
	return r
}

// Previous returns the previous rune from the input and moves the current position of
// the lexer behind.
func (l *Lexer) Previous() rune {
	l.CurrentPosition -= RunePosition(l.CurrentRuneWidth)
	r, _ := utf8.DecodeRuneInString(l.Input[l.CurrentPosition:])
	return r
}

// Ignore skips and returns the next rune from the input.
func (l *Lexer) Ignore() rune {
	r := l.Next()
	l.startPosition = l.CurrentPosition
	return r
}

// IgnoreUpTo skips runes from the input and returns the rune last seen by the predicate.
//
// Returns EOF if the end of input is encountered before the predicate is satisfied.
func (l *Lexer) IgnoreUpTo(predicate RunePredicate) rune {
	return l.consumeUpTo(predicate, l.Ignore)
}

// Emit emits a token of the specified type.
func (l *Lexer) Emit(tokenType TokenType) {
	t := Token{tokenType, l.Input[l.startPosition:l.CurrentPosition]}
	l.tokens <- t
	l.tokenMutex.Lock()
	l.previousToken = l.currentToken
	l.currentToken = t
	l.tokenMutex.Unlock()
	l.startPosition = l.CurrentPosition
}

// Errorf emits an error token with the specified error message as its value.
func (l *Lexer) Errorf(format string, args ...interface{}) StateFunc {
	l.tokens <- Token{TokenError, fmt.Sprintf(format, args...)}
	return nil
}

func (l *Lexer) consumeUpTo(predicate RunePredicate, consumer func() rune) rune {
	var r rune
	for {
		r = l.Peek()
		if predicate(r) || r == EOF {
			break
		}
		r = consumer()
	}
	return r
}
