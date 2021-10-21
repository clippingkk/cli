package core

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

//go:embed clippings_en.txt
var clippingENFile []byte

type clippingFileParserTestSuite struct {
	suite.Suite
}

func (s *clippingFileParserTestSuite) SetupTest() {
}

func (s *clippingFileParserTestSuite) TestParserENFile() {
	pser := NewClippingParser(clippingENFile)
	err := pser.Prepare()
	assert.Nil(s.T(), err)
	result, err := pser.DoParse()
	assert.Nil(s.T(), err)
	assert.Len(s.T(), result, 1)
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(clippingFileParserTestSuite))
}
