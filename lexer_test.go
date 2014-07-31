package lexer_test

import (
	"unicode"

	"github.com/eczarny/lexer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const Token lexer.TokenType = iota

var _ = Describe("Lexer", func() {
	numeric := func(r rune) bool {
		return r == '.' || unicode.IsDigit(r)
	}

	assertToken := func(token lexer.Token, tokenType lexer.TokenType, tokenValue interface{}) {
		Expect(token).To(Equal(lexer.Token{tokenType, tokenValue}))
	}

	It("should return the next token emitted by the lexer (i.e. NextToken and Emit)", func() {
		l := lexer.NewLexer("E = m * c^2", func(l *lexer.Lexer) lexer.StateFunction {
			l.Next()
			l.Emit(Token)
			return nil
		})
		assertToken(l.NextToken(), Token, "E")
	})

	It("should return the next rune from the input and moves the current position of the lexer ahead (i.e. Next)", func(done Done) {
		r := make(chan rune)
		p := make(chan lexer.RunePosition)
		l := lexer.NewLexer("a^2 + b^2 = c^2", func(l *lexer.Lexer) lexer.StateFunction {
			p <- l.CurrentPosition
			r <- l.Next()
			p <- l.CurrentPosition
			r <- l.Next()
			p <- l.CurrentPosition
			r <- l.Next()
			p <- l.CurrentPosition
			l.Emit(Token)
			return nil
		})
		Expect(<-p).To(Equal(lexer.RunePosition(0)))
		Expect(<-r).To(Equal('a'))
		Expect(<-p).To(Equal(lexer.RunePosition(1)))
		Expect(<-r).To(Equal('^'))
		Expect(<-p).To(Equal(lexer.RunePosition(2)))
		Expect(<-r).To(Equal('2'))
		Expect(<-p).To(Equal(lexer.RunePosition(3)))
		assertToken(l.NextToken(), Token, "a^2")
		close(done)
	})

	It("should return the rune last seen by the predicate and moves the current position of the lexer ahead (i.e. NextUpTo)", func(done Done) {
		r := make(chan rune)
		p := make(chan lexer.RunePosition)
		l := lexer.NewLexer("3.14", func(l *lexer.Lexer) lexer.StateFunction {
			p <- l.CurrentPosition
			r <- l.NextUpTo(func(r rune) bool {
				return !numeric(r)
			})
			p <- l.CurrentPosition
			l.Emit(Token)
			return nil
		})
		Expect(<-p).To(Equal(lexer.RunePosition(0)))
		Expect(<-r).To(Equal(lexer.EOF))
		Expect(<-p).To(Equal(lexer.RunePosition(4)))
		assertToken(l.NextToken(), Token, "3.14")
		close(done)
	})

	It("should return the next rune from the input without moving the current position of the lexer ahead (i.e. Peek)", func(done Done) {
		r := make(chan rune)
		p := make(chan lexer.RunePosition)
		l := lexer.NewLexer("E = m * c^2", func(l *lexer.Lexer) lexer.StateFunction {
			p <- l.CurrentPosition
			r <- l.Peek()
			p <- l.CurrentPosition
			l.Emit(Token)
			return nil
		})
		Expect(<-p).To(Equal(lexer.RunePosition(0)))
		Expect(<-r).To(Equal('E'))
		Expect(<-p).To(Equal(lexer.RunePosition(0)))
		assertToken(l.NextToken(), Token, "")
		close(done)
	})

	It("should return the previous rune from the input and moves the current position of the lexer behind (i.e. Previous)", func(done Done) {
		r := make(chan rune)
		p := make(chan lexer.RunePosition)
		l := lexer.NewLexer("C = 2 * Pi * r", func(l *lexer.Lexer) lexer.StateFunction {
			p <- l.CurrentPosition
			r <- l.Next()
			p <- l.CurrentPosition
			l.Emit(Token)
			r <- l.Previous()
			p <- l.CurrentPosition
			return nil
		})
		Expect(<-p).To(Equal(lexer.RunePosition(0)))
		Expect(<-r).To(Equal('C'))
		Expect(<-p).To(Equal(lexer.RunePosition(1)))
		assertToken(l.NextToken(), Token, "C")
		Expect(<-r).To(Equal('C'))
		Expect(<-p).To(Equal(lexer.RunePosition(0)))
		close(done)
	})

	It("should skip and return the next rune from the input (i.e. Ignore)", func(done Done) {
		r := make(chan rune)
		l := lexer.NewLexer("e = 2.71", func(l *lexer.Lexer) lexer.StateFunction {
			r <- l.Next()
			l.Emit(Token)
			r <- l.Ignore()
			r <- l.Next()
			l.Emit(Token)
			r <- l.Ignore()
			l.NextUpTo(func(r rune) bool {
				return !numeric(r)
			})
			l.Emit(Token)
			return nil
		})
		Expect(<-r).To(Equal('e'))
		assertToken(l.NextToken(), Token, "e")
		Expect(<-r).To(Equal(' '))
		Expect(<-r).To(Equal('='))
		assertToken(l.NextToken(), Token, "=")
		Expect(<-r).To(Equal(' '))
		assertToken(l.NextToken(), Token, "2.71")
		close(done)
	})

	It("should skip runes from the input and return the rune last seen by the predicate (i.e. IgnoreUpTo)", func(done Done) {
		r := make(chan rune)
		p := make(chan lexer.RunePosition)
		l := lexer.NewLexer("E = m * c^2", func(l *lexer.Lexer) lexer.StateFunction {
			r <- l.IgnoreUpTo(func(r rune) bool {
				return r == 'c'
			})
			p <- l.CurrentPosition
			r <- l.Next()
			p <- l.CurrentPosition
			r <- l.Next()
			p <- l.CurrentPosition
			r <- l.Next()
			p <- l.CurrentPosition
			l.Emit(Token)
			return nil
		})
		Expect(<-r).To(Equal('c'))
		Expect(<-p).To(Equal(lexer.RunePosition(8)))
		Expect(<-r).To(Equal('c'))
		Expect(<-p).To(Equal(lexer.RunePosition(9)))
		Expect(<-r).To(Equal('^'))
		Expect(<-p).To(Equal(lexer.RunePosition(10)))
		Expect(<-r).To(Equal('2'))
		Expect(<-p).To(Equal(lexer.RunePosition(11)))
		assertToken(l.NextToken(), Token, "c^2")
		close(done)
	})

	It("should return the most recently emitted token (i.e. PreviousToken)", func(done Done) {
		l := lexer.NewLexer("a^2 + b^2 = c^2", func(l *lexer.Lexer) lexer.StateFunction {
			p := func(r rune) bool {
				return !(r == 'a' || r == 'b' || r == 'c' || r == '^' || r == '2')
			}
			l.NextUpTo(p)
			l.Emit(Token)
			l.Ignore()
			l.Next()
			l.Emit(Token)
			l.Ignore()
			l.NextUpTo(p)
			l.Emit(Token)
			l.Ignore()
			l.Next()
			l.Emit(Token)
			l.Ignore()
			l.NextUpTo(p)
			l.Emit(Token)
			return nil
		})
		assertToken(l.PreviousToken(), Token, nil)
		assertToken(l.NextToken(), Token, "a^2")
		assertToken(l.PreviousToken(), Token, nil)
		assertToken(l.NextToken(), Token, "+")
		assertToken(l.PreviousToken(), Token, "a^2")
		assertToken(l.NextToken(), Token, "b^2")
		assertToken(l.PreviousToken(), Token, "+")
		assertToken(l.NextToken(), Token, "=")
		assertToken(l.PreviousToken(), Token, "b^2")
		assertToken(l.NextToken(), Token, "c^2")
		assertToken(l.PreviousToken(), Token, "=")
		close(done)
	})

	It("should emit an error token with the specified error message as its value (i.e. Errorf)", func() {
		l := lexer.NewLexer("E = m * c^2", func(l *lexer.Lexer) lexer.StateFunction {
			return l.Errorf("Unexpected input")
		})
		assertToken(l.NextToken(), lexer.TokenError, "Unexpected input")
	})
})
