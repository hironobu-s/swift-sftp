package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/urfave/cli"
)

func listContainer(ctx *cli.Context) (err error) {
	c := Config{}
	if err = c.LoadFromContext(ctx); err != nil {
		return err
	}

	s := NewSwift(c)
	if err = s.Init(); err != nil {
		return err
	}

	list, err := s.ListContainer()
	if err != nil {
		return err
	}

	var w1, w2, w3 int
	for _, info := range list {
		var l int

		l = len(info.Name)
		if l > w1 {
			w1 = l
		}

		l = len(strconv.FormatInt(info.Count, 10))
		if l > w2 {
			w2 = l
		}

		l = len(strconv.FormatInt(info.Bytes, 10))
		if l > w3 {
			w3 = l
		}
	}

	if w1 < len("[Name]") {
		w1 = len("[Name]")
	}
	if w2 < len("[Objects]") {
		w2 = len("[Objects]")
	}
	if w3 < len("[Total]") {
		w3 = len("[Total]")
	}

	w1 += 1
	w2 += 1
	w3 += 1

	format := "%-" + strconv.Itoa(w1) + "s % " + strconv.Itoa(w2) + "s % " + strconv.Itoa(w3) + "s\n"
	fmt.Fprintf(os.Stdout, format, "[Name]", "[Objects]", "[Total]")
	for _, info := range list {
		fmt.Fprintf(os.Stdout, format,
			info.Name,
			strconv.FormatInt(info.Count, 10),
			strconv.FormatInt(info.Bytes, 10))
	}

	return nil
}

func createContainer(ctx *cli.Context) (err error) {
	c := Config{}
	if err = c.LoadFromContext(ctx); err != nil {
		return err
	}

	if ctx.NArg() == 0 {
		return errors.New("Container name required")
	}
	c.Container = ctx.Args()[0]

	s := NewSwift(c)
	if err = s.Init(); err != nil {
		return err
	}

	exists, err := s.ExistsContainer()
	if err != nil {
		return err
	} else if exists {
		log.Warnf("Container '%s' is already exists", c.Container)
		return nil
	}

	if err = s.CreateContainer(); err != nil {
		return err
	}

	log.Infof("Container '%s' created", c.Container)
	return nil
}

func deleteContainer(ctx *cli.Context) (err error) {
	c := Config{}
	if err = c.LoadFromContext(ctx); err != nil {
		return err
	}

	if ctx.NArg() == 0 {
		return errors.New("Container name required")
	}
	c.Container = ctx.Args()[0]

	fmt.Fprintf(os.Stderr, "[CAUTION] Container '%s' will be deleted. Are you sure? [Y/n] ", c.Container)

	buf := make([]byte, 0, 10)
	_, err = fmt.Fscanln(os.Stdin, &buf)
	if err != nil {
		return err
	} else if string(buf) != "Y" {
		return nil
	}

	s := NewSwift(c)
	if err = s.Init(); err != nil {
		return err
	}

	if err = s.DeleteContainer(); err != nil {
		return err
	}

	log.Infof("Container '%s' deleted", c.Container)
	return nil
}
