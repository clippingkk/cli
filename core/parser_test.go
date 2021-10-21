package core

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

//go:embed clippings_en.txt
var clippingENFile []byte

//go:embed clippings_en.result.json
var enDistResult string

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
	assert.Len(s.T(), result, 40)

	buf, _ := json.Marshal(result)
	assert.JSONEq(s.T(), enDistResult, string(buf))
}

//go:embed clippings_other.txt
var clippingOtherFile []byte

//go:embed clippings_en.result.json
var otherDistResult string

func (s *clippingFileParserTestSuite) TestParserOtherFile() {
	pser := NewClippingParser(clippingOtherFile)
	err := pser.Prepare()
	assert.Nil(s.T(), err)
	result, err := pser.DoParse()
	assert.Nil(s.T(), err)
	assert.Len(s.T(), result, 40)

	buf, _ := json.Marshal(result)
	assert.JSONEq(s.T(), otherDistResult, string(buf))
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(clippingFileParserTestSuite))
}