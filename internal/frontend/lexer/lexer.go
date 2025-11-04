package lexer

// This file defines the stateless lexer interface that will be called by the pipeline.
// The actual implementation is in tokenizer.go
//
// NOTE: We cannot import context package here due to import cycle.
// The pipeline will call LexFile which uses the existing tokenizer.
