package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Define command-line arguments
	input := flag.String("input", "", "Input files/application, comma separated")
	// Add more flags as necessary...

	flag.Parse()
	filesList := strings.Split(*input, ",")
	fmt.Printf("Input files: %v\n", filesList)

	// Validate inputs
	if *input == "" {
		log.Fatal("Input and output paths are required")
	}

	// Create a temporary directory for building
	buildDir, err := os.MkdirTemp("", "build")
	if err != nil {
		log.Fatalf("Error creating temporary build directory: %v", err)
	}
	defer os.RemoveAll(buildDir)

	for _, file := range filesList {
		// Copy files from input to build directory
		fmt.Printf("Copying %s to %s\n", file, buildDir)
		fileInfo, err := os.Stat(file)
		if err != nil {
			// handle the error, e.g., file does not exist or other issues
			log.Fatalf("Er ror fileInfo from os.Stat: %v", err)
		}
		if !fileInfo.IsDir() {
			err = copyFile(file, filepath.Join(buildDir, filepath.Base(file)))
			if err != nil {
				log.Fatalf("Error copying argument files: %v", err)
			}
			continue
		}
		// Copy files from input to build directory
		err = filepath.WalkDir(file, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(file, path)
			if err != nil {
				return err
			}

			targetPath := filepath.Join(buildDir, relPath)

			if d.IsDir() {
				return os.MkdirAll(targetPath, 0755)
			}

			return copyFile(path, targetPath)
		})
	}

	if err != nil {
		log.Fatalf("Error copying files: %v", err)
	}

	// Create tarball from build directory
	err = createTarball(buildDir, "./build.tar.gz")
	if err != nil {
		log.Fatalf("Error creating tarball: %v", err)
	}

	fmt.Printf("Build completed: %s\n", "./build.tar.gz")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

// Compress a directory to tar.gz format
func createTarball(srcDir, targetTarGz string) error {
	// Create output file
	tarGzFile, err := os.Create(targetTarGz)
	if err != nil {
		return err
	}
	defer tarGzFile.Close()

	// Create a gzip writer
	gzipWriter := gzip.NewWriter(tarGzFile)
	defer gzipWriter.Close()

	// Create a tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk through the source directory and add files to the tar
	return filepath.Walk(srcDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a header
		header, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}

		// Update the Name in the header to not include the srcDir
		relativePath, err := filepath.Rel(srcDir, file)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relativePath) // This changes the path in the tar

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// If not a directory, write file content
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			defer data.Close()

			if _, err := io.Copy(tarWriter, data); err != nil {
				return err
			}
		}

		return nil
	})
}
