package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const imageName = "opencv"

var cudaVersion = "10.0"
var osName = "ubuntu16.04"

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func buildImage(dockerfile, logPath, imageName string) error {
	// open file for logging
	if exists(logPath) {
		os.Remove(logPath)
	}

	logFile, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer logFile.Close()

	// build
	cmd := exec.Command("docker", "build", "-f", dockerfile, "-t", imageName, ".")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("opening stdout pipe: %w", err)
	}
	cmd.Start()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		logFile.Write(scanner.Bytes())
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("building dockerfile: %w", err)
	}

	return nil
}

func generateDockerfile(opencvVersion string) string {
	dependencies := []string{
		"build-essential",
		"cmake",
		"wget",
		"unzip",
		"libgtk2.0-dev",
		"pkg-config",
		"libavcodec-dev",
		"libavformat-dev",
		"libswscale-dev",
		"libpq-dev",
		"python-dev",
		"python-numpy",
		"python3-dev",
		"python3-numpy",
		"libtbb2",
		"libtbb-dev",
		"libjpeg-dev",
		"libpng-dev",
		"libtiff-dev",
		"libjasper-dev",
		"libdc1394-22-dev",
		"libavformat-dev",
		"libtheora-dev",
		"libvorbis-dev",
		"libxvidcore-dev",
		"libx264-dev",
		"yasm",
		"libopencore-amrnb-dev",
		"libopencore-amrwb-dev",
		"libv4l-dev",
		"libxine2-dev",
		"libgstreamer1.0-dev",
		"libgstreamer-plugins-base1.0-dev",
		"libeigen3-dev",
		"libglew-dev",
		"libtiff5-dev",
		"zlib1g-dev",
		"libpng12-dev",
		"libavformat-dev",
		"libavutil-dev",
		"libpostproc-dev",
		"libvtk6-dev",
	}

	opencvBuildOptions := []string{
		"CMAKE_BUILD_TYPE=Release",
		"CMAKE_INSTALL_PREFIX=/usr/local",
		"BUILD_EXAMPLES=OFF",
		"WITH_TBB=ON",
		"WITH_IPP=ON",
		"FORCE_VTK=ON",
		"WITH_V4L=ON",
		"WITH_XINE=ON",
		"WITH_GDAL=ON",
		"WITH_OPENCL=ON",
		"WITH_OPENGL=ON",
		"BUILD_opencv_cudacodec=OFF",
		"ENABLE_FAST_MATH=ON",
		"CUDA_FAST_MATH=ON",
		"WITH_CUDA=ON",
		"CUDA_ARCH_BIN='3.0 3.5 3.7 5.0 5.2 6.0 6.1 6.2 7.0 7.5'",
		"CUDA_ARCH_PTX='3.0 3.5 3.7 5.0 5.2 6.0 6.1 6.2 7.0 7.5'",
		"OPENCV_DNN_CUDA=OFF",
		"WITH_CUBLAS=ON",
		"WITH_CUFFT=ON",
		"WITH_EIGEN=ON",
		"EIGEN_INCLUDE_PATH=/usr/include/eigen3",
	}

	lines := []string{}
	addLine := func(line string) {
		lines = append(lines, line)
	}

	addLine(fmt.Sprintf("FROM nvidia/cuda:%s-cudnn7-devel-%s", cudaVersion, osName))
	addLine(`ENV DEBIAN_FRONTEND noninteractive`)
	addLine(fmt.Sprintf("ARG OPENCV_VERSION='%s'", opencvVersion))

	// install dependencies
	addLine(`RUN apt-get update -y && apt-get install -y --no-install-recommends \`)
	for i, dep := range dependencies {
		if i == len(dependencies)-1 {
			addLine(fmt.Sprintf("\t%s", dep))
			break
		}

		addLine(fmt.Sprintf("\t%s \\", dep))
	}

	// download opencv
	addLine(`WORKDIR /opt`)
	addLine(`RUN wget https://github.com/opencv/opencv/archive/${OPENCV_VERSION}.zip && \`)
	addLine(`unzip ${OPENCV_VERSION}.zip && rm ${OPENCV_VERSION}.zip && \`)
	addLine(`mv opencv-${OPENCV_VERSION} opencv && \`)
	addLine(`wget https://github.com/opencv/opencv_contrib/archive/${OPENCV_VERSION}.zip && \`)
	addLine(`unzip ${OPENCV_VERSION}.zip && rm ${OPENCV_VERSION}.zip && \`)
	addLine(`mv opencv_contrib-${OPENCV_VERSION} opencv/opencv_contrib && \`)
	addLine(`mkdir /opt/opencv/build && cd /opt/opencv/build && \`)

	// build opencv
	addLine(`cmake \`)
	for _, opt := range opencvBuildOptions {
		addLine(fmt.Sprintf("\t-D %s \\", opt))
	}
	addLine(`.. && \`)
	addLine(`make install && \`)
	addLine(`make clean`)

	return strings.Join(lines, "\n")
}

func main() {
	opencvVersions := []string{
		"3.0.0",
		"3.1.0",
		"3.2.0",
		"3.3.0",
		"3.3.1",
		"3.4.0",
		"3.4.2",
		"3.4.3",
		"3.4.4",
		"3.4.5",
		"3.4.8",
		"3.4.9",
		"4.0.0",
		"4.0.1",
		"4.1.0",
		"4.1.1",
		"4.2.0",
	}

	for _, dirName := range []string{"dockerfiles", "logs"} {
		if !exists(dirName) {
			err := os.Mkdir(dirName, 0777)
			if err != nil {
				fmt.Println(fmt.Errorf("creating %s directory: %w", dirName, err))
				os.Exit(1)
			}
			fmt.Printf("created %s dir\n", dirName)
			os.Exit(1)
		}
	}

	for _, opencvVersion := range opencvVersions {
		// generate dockerfile
		dockerfile := generateDockerfile(opencvVersion)

		tag := fmt.Sprintf("%s-cuda%s-%s", opencvVersion, cudaVersion, osName)
		path := filepath.Join("dockerfiles", tag)
		f, err := os.Create(path)
		if err != nil {
			fmt.Println(fmt.Errorf("creating dockerfile: %w", err))
			os.Exit(1)
		}
		f.Write(([]byte)(dockerfile))
		f.Close()

		// build image
		log.Printf("building %s... ", tag)
		target := fmt.Sprintf("%s:%s", imageName, tag)
		err = buildImage(path, fmt.Sprintf("logs/%s.txt", tag), target)
		if err != nil {
			log.Printf("failed\n")
		} else {
			log.Printf("ok\n")
		}
	}
}
