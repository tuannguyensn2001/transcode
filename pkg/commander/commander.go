package commander

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

func split(data []byte, atEOF bool) (advance int, token []byte, spliterror error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	//windows \r\n
	//so  first \r and then \n can remove unexpected line break
	if i := bytes.IndexByte(data, '\r'); i >= 0 {
		// We have a cr terminated line
		return i + 1, data[0:i], nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil
}

type Commander struct {
	command   string
	args      []string
	outStream io.ReadCloser
	errStream io.ReadCloser
	cmd       *exec.Cmd
	isRunning bool
}

func New(command string, args ...string) Commander {
	return Commander{
		command: command,
		args:    args,
	}
}

func (c *Commander) IsRunning() bool {
	return c.isRunning
}

func (c *Commander) SetArgs(args []string) {
	c.args = args
}

func (c *Commander) Run() chan error {
	done := make(chan error, 1)
	if c.command == "" {
		done <- errors.New("cannot run without command")
		close(done)
		return done
	}
	if len(c.args) == 0 {
		done <- errors.New("cannot run command with no argument")
		close(done)
		return done
	}

	cmd := exec.Command(c.command, c.args...)
	errStream, err := cmd.StderrPipe()
	if err != nil {
		done <- err
		close(done)
		return done
	}
	c.errStream = errStream

	outStream, err := cmd.StdoutPipe()
	if err != nil {
		done <- err
		close(done)
		return done
	}
	c.outStream = outStream

	err = cmd.Start()
	c.cmd = cmd
	go func(err error) {
		c.isRunning = true
		defer func() {
			c.isRunning = false
		}()
		if err != nil {
			var outb bytes.Buffer
			io.Copy(&outb, outStream)
			done <- fmt.Errorf("failed start %s (%s) with %s, message %s", c.command, c.args, err, outb.String())
			close(done)
			return
		}

		err = cmd.Wait()

		if err != nil {
			if err.Error() != "signal: killed" {
				var outb bytes.Buffer
				io.Copy(&outb, outStream)
				err = fmt.Errorf("failed finish %s (%s) with %s message %s", c.command, c.args, err, outb.String())
			} else {
				err = nil
			}
		}
		done <- err
		close(done)
	}(err)

	return done
}

func (c *Commander) Stop() error {
	return c.cmd.Process.Kill()
}

func (c *Commander) StderrLogs() chan string {
	out := make(chan string)

	go func() {
		defer close(out)
		if c.errStream == nil {
			out <- ""
			return
		}
		defer c.errStream.Close()

		scanner := bufio.NewScanner(c.errStream)

		scanner.Split(split)
		buf := make([]byte, 2)
		scanner.Buffer(buf, bufio.MaxScanTokenSize)

		for scanner.Scan() {
			line := scanner.Text()
			out <- line
		}
	}()

	return out
}

func (c *Commander) StdoutLogs() chan string {
	out := make(chan string)

	go func() {
		defer close(out)
		if c.outStream == nil {
			out <- ""
			return
		}
		defer c.outStream.Close()

		scanner := bufio.NewScanner(c.outStream)

		scanner.Split(split)
		buf := make([]byte, 2)
		scanner.Buffer(buf, bufio.MaxScanTokenSize)

		for scanner.Scan() {
			line := scanner.Text()
			out <- line
		}
	}()

	return out
}
