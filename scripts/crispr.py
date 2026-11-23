# read --input and --out arguments
import argparse

parser = argparse.ArgumentParser()
parser.add_argument("--file", required=True, help="Input file")
parser.add_argument("--out", required=True, help="Output file")
args = parser.parse_args()

input_file = args.file
output_file = args.out

# Implement the actual processing logic here
# For now, just copy the input file to the output file
#import shutil

#shutil.copy(input_file, output_file)

# write test into file
with open(output_file, "w") as f:
    f.write("test")