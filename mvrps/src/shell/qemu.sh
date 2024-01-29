#!/bin/bash

# Script to convert input files to a QEMU-compatible image using qemu-img

# Check if qemu-img is installed
if ! command -v qemu-img &> /dev/null; then
    echo "Error: qemu-img is not installed. Please install QEMU."
    exit 1
fi

# Check if at least one input file is provided
if [ "$#" -lt 1 ]; then
    echo "Usage: $0 input_file [output_format] [output_file]"
    exit 1
fi

input_file="../openwrt-23.05.2-x86-64-rootfs.tar.gz"

# Set default output format to qcow2
output_format="qcow2"

# Set default output file to input_file.qcow2
output_file="${input_file%.*}.qcow2"

# Check if output format is specified
if [ "$#" -ge 2 ]; then
    output_format="$1"
fi

# Check if output file is specified
if [ "$#" -ge 3 ]; then
    output_file="$2"
fi

# Check if the input file exists
if [ ! -f "$input_file" ]; then
    echo "Error: Input file '$input_file' not found."
    exit 1
fi

# Perform the conversion using qemu-img
echo "Converting $input_file to $output_format format..."
qemu-img convert -f raw -O "$output_format" "$input_file" "$output_file"

# Check if conversion was successful
if [ $? -eq 0 ]; then
    echo "Conversion successful. Output file: $output_file"
else
    echo "Error: Conversion failed."
fi
