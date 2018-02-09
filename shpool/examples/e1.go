package main

import "github.com/brentp/go-athenaeum/shpool"

func main() {
	opts := shpool.Options{LogPrefix: ""}
	p := shpool.New(4, nil, &opts)
	p.Add(shpool.Process{Command: "echo hello && sleep 3 && echo goodbye", Prefix: "sleep", CPUs: 4})
	p.Add(shpool.Process{Command: "echo hello && sleep 3 && echo goodbye", Prefix: "sleep", CPUs: 2})
	p.Add(shpool.Process{Command: "echo hello && sleep 3 && echo goodbye", Prefix: "sleep"})
	p.Add(shpool.Process{Command: "echo hello && sleep 3 && echo goodbye", Prefix: "sleep"})
	p.Add(shpool.Process{Command: "ls -lh", Prefix: "ls"})
	p.Add(shpool.Process{Command: "ls -lh ../", Prefix: "ls ../"})
	p.Add(shpool.Process{Command: "ls -lh xxx", Prefix: "ls xxx"})
	p.Add(shpool.Process{Command: "echo hello && sleep 6 && echo goodbye", Prefix: "sleep"})

	//time.Sleep(1 * time.Second)

	//p.KillAll()

	if err := p.Wait(); err != nil {
		panic(err)
	}

}
