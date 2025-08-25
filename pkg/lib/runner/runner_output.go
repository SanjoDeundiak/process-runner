package runner

func (runner *Runner) Output(id string) (<-chan []byte, <-chan []byte, error) {
	pe, err := runner.getProcess(id)
	if err != nil {
		return nil, nil, err
	}

	logger.Printf("Start subscription to stdout for %s", id)
	stdoutCh := pe.stdout.Subscribe(5)
	logger.Printf("Subscribed to stdout for %s", id)

	logger.Printf("Start subscription to stderr for %s", id)
	stderrCh := pe.stderr.Subscribe(5)
	logger.Printf("Subscribed to stderr for %s", id)

	return stdoutCh, stderrCh, nil
}
