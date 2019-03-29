This is an example of creating a deployment package with multiple
files including external libraries via composer.

### Create an environment

```
fission env create --name php --image fission/php-env:latest --builder fission/php-builder:latest --version 2
```

### Create a zip file with all your files

```
zip -r multifile.zip . -i *.php *.txt composer.json
```

### Create a package
```
fission package create --sourcearchive multifile.zip --env php
```
This command will print the created package. We will use it in the next step.


### Create a function

Since there are multiple files, you have to specify an _entrypoint_ to
for the function.  Its format is `<file path>::<function name>`. In our
example, that's `handlers/FileReader.php::execute`, to run function `execute` in `handlers/FileReader.php`.

```
fission function create --name multifile --env php --pkg <created-pkg-name> --entrypoint "handlers/FileReader.php::execute"
```

### Test it

```
fission function test --name multifile
```

You should see the "Hello, world" message.


## Updating the function

### Edit a file

```
echo "I said hellooooo!" > message.txt
```

### Update the deployment package

```
zip -r multifile.zip . -i *.php *.txt composer.json
```

### Update the package

```
fission package update --name <created-pkg-name> --sourcearchive multifile.zip --env php
```

### Test it

```
fission function test --name multifile
```

You should now see your new, edited message.

