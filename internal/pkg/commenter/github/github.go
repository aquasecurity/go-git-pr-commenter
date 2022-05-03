package github

type Github struct{}

//TODO implement here

func NewGithub() *Github {
	return &Github{}
}

func (c *Github) WriteMultiLineComment(file, comment, startLine, endLine string) error {
	return nil
}

func (c *Github) WriteLineComment(file, comment, line string) error {
	return nil
}
