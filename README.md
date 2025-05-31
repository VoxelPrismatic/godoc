# godoc

Like `go doc` but with colors.

## Installation

```
$ go install github.com/VoxelPrismatic/godoc@latest
```

## Usage

Accepts all the arguments and flags `go doc` works with. Godoc is just a simple wrapper around the go doc tool.

Example:

```sh
$ godoc io.Writer
```

![godoc sample 1](./samples/io.Writer.png)

```sh
$ godoc os.WriteFile
```

![godoc sample 2](./samples/os.WriteFile.png)

```sh
$ godoc github.com/tree-sitter/go-tree-sitter.CaptureQuantifierZero
```

![godoc sample 3](./samples/go-tree-sitter.CaptureQuantifierZero.png)

```sh
$ godoc fs.FileInfo
```

![godoc sample 4](./samples/fs.FileInfo.png)

## Styling

Unlike other forks of `godocc`, this uses TreeSitter for the sole purpose
of using your terminal's highlighting. Absolutely no color or styling customization
is provided in the command line; just your standard ANSI colors.
