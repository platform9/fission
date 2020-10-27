# Fission CLI Extensibility

### Approach
To sum up the approach: git-style plugins.

Plugins are named with the fission prefix `fission-*`. When fission is invoked with an undefined/non-core subcommand 
is called (like `fission foo`). Fission will look in the PATH  

**Installing Fission**
```bash
# The same as before
$ curl -Lo fission https://github.com/fission/fission/releases/download/0.7.2/fission-cli-osx && chmod +x fission && sudo mv fission /usr/local/bin/
```

**Installing Fission Workflows**
```bash
# The same process as Fission cli itself
$ curl -Lo fission https://github.com/fission/fission-workflows/releases/download/0.4.0/fission-workflows-osx && chmod +x fission-workflows && sudo mv fission-workflows /usr/local/bin/
```

**Invoking Fission Workflows**
```bash
$ fission workflows invocation get b1278e802a
# Which is equivalent to:
$ fission-workflows invocation get b1278e802a
```
General flow: 
1. fission does not recognize `workflows` subcommand
2. fission checks the path for a binary called `fission-workflows`
3. fission finds the binary.
4. fission invokes the `fission-workflows`, passing the remainder of the arguments.

**Discoverability: fission --help**
```bash
$ fission --help
USAGE:
   fission [global options] command [command options] [arguments...]

VERSION:
   0.6.0

COMMANDS:
     function, fn                  Create, update and manage functions
     httptrigger, ht, route        Manage HTTP triggers (routes) for functions
     timetrigger, tt, timer        Manage Time triggers (timers) for functions
     mqtrigger, mqt, messagequeue  Manage message queue triggers for functions
     environment, env              Manage environments
     watch, w                      Manage watches
     package, pkg                  Manage packages
     spec, specs                   Manage a declarative app specification
     upgrade                       Upgrade tool from fission v0.1
     tpr2crd                       Migrate tool for TPR to CRD
     help, h                       Shows a list of commands or help for one command

PLUGINS:
     workflows, wf                 Inspect and manage workflow executions
     ui                            Start the user interface

GLOBAL OPTIONS:
   --server value  Fission server URL (default: "http://127.0.0.1:65356")
   --help, -h      show help
   --version, -v   print the version
```
Of course Fission needs to be able to find all plugins for this. There are several ways in which we can provide discoverability. The simplest one is for Fission to look in the path for all binaries starting with the `fission-*` prefix. Optionally, fission could invoke a specific command on the subcommand to get info about the plugin (such as version, help text, aliases...)

With Fission Workflows this info would look something like this: 
```bash
$ fission-workflows --plugin
name: workflows
version: 0.4.0
help: Inspect and manage workflow executions
```
The idea is that this plugin info is all completely optional. 
If it is not available, we simply degrade the results to user. 
This way users/we can easily prototype or add plugins without having to worry about adhering to some interface.

**List version**
```bash
$ fission --version
client: 
  fission: 0.8.0
  fission-workflows: 0.4.0
server:
  fission: 0.8.1
  fission-workflows: 0.3.0
```
Again, versioning info for fission-workflows is taken from the plugin info of the commands. 
Note: a related issue is to have some more formalized plugin support/discoverability on the server-side, 
but that is out of the scope of this issue.

### Other (optional) extensions and notes
- Like git we could setup a preferred binary path, where fission looks first when searching for the subcommand. 
This could optionally be defined with a `FISSION_EXEC_PATH`.
- With the current approach we cannot have aliases for commands---fission will not be able to find fission-workflows 
when the user calls `fission wf`. This might be UX issue, with these long path names. One option is let the user fix 
it themselves by symlinking `fission-wf` to `fission-workflows`; using the plugin info Fission can recognize and 
merge aliases together.
- To help detect versioning conflicts (old version of fission, too new version of fission workflows). We could add 
a `requires` field to the fission-workflows plugin info. Then we could throw a warning or error, when two out of sync 
versions are being used. 
- To avoid unhelpful errors to the user when they have not installed a plugin, we could add a heuristic to check 
`https://github.com/fission/SUBCOMMAND` to see if the subcommand might be an uninstalled plugin. 
OR, we could lookup a simple text file that contains common plugins `https://github.com/fission/fission/plugins.txt` 
and list them as suggestions to the user. OR we could of course just default to a bit help text that says something 
like `unknown subcommand 'foo'. If this is a plugin, ensure that it is present on your PATH`.

---

### Motivation

The proposed approach is to use the git-based plugin system for now. Reasons for this approach over a sophisticated, 
integrated plugin-based approach:
- It is low effort to implement.
- It is easy to extend with minimal to no required interface.
- The binaries remain standalone, allowing users to separate them if needed and make independent development on the 
binaries easy.

Limitations of the proposed approach:
- The user still has to do some work, adding binaries to the PATH; ensuring that permissions are correct; ensuring 
that the binary is executable; how to deal with duplicate binaries on the PATH. All this makes this approach assume 
basic/intermediate knowledge of the OS from the user.
- I have to admit: I am not entirely sure if this approach requires any changes for Windows. Probably not.
- Upgrading fission with many plugins could be cumbersome, as you would need to upgrade each binary one by one. 
Improving this is probably best left to future work.


The more heavyweight solution solves some of these limitations to an extent, but these do not way up to the increased 
development and maintenance cost IMO. If needed we could explore this option (or some hybrid option) in the future.
