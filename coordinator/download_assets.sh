#!/bin/sh
apt update
apt install wget libdigest-sha-perl -y

RELEASE_VERSION_HI=v0.12.0
RELEASE_VERSION_LO=v0.11.4

P_CHECKSUMS=$(wget -O- https://circuit-release.s3.us-west-2.amazonaws.com/setup/sha256sum)
DOWNLOAD_RESULT=$?
ERROR=$(echo "$P_CHECKSUMS" | grep "Error")

if [ $DOWNLOAD_RESULT -ne 0 ] || [ "$ERROR" != "" ]; then
	echo "Failed to download params checksums"
	echo "$P_CHECKSUMS"
	exit 1
fi

R_CHECKSUMS_HI=$(wget -O- https://circuit-release.s3.us-west-2.amazonaws.com/release-$RELEASE_VERSION_HI/sha256sum)
DOWNLOAD_RESULT=$?
ERROR=$(echo "$R_CHECKSUMS_HI" | grep "Error")
if [ $DOWNLOAD_RESULT -ne 0 ] || [ "$ERROR" != "" ]; then
	echo "Failed to download release checksum for $RELEASE_VERSION_HI"
	echo "$R_CHECKSUMS_HI"
	exit 1
fi

R_CHECKSUMS_LO=$(wget -O- https://circuit-release.s3.us-west-2.amazonaws.com/release-$RELEASE_VERSION_LO/sha256sum)
DOWNLOAD_RESULT=$?
ERROR=$(echo "$R_CHECKSUMS_LO" | grep "Error")
if [ $DOWNLOAD_RESULT -ne 0 ] || [ "$ERROR" != "" ]; then
	echo "Failed to download release checksum for $RELEASE_VERSION_LO"
	echo "$R_CHECKSUMS_LO"
	exit 1
fi

PARAMS20_SHASUM=$(echo "$P_CHECKSUMS" | grep "params20" | cut -d " " -f 1)
PARAMS21_SHASUM=$(echo "$P_CHECKSUMS" | grep "params21" | cut -d " " -f 1)
PARAMS24_SHASUM=$(echo "$P_CHECKSUMS" | grep "params24" | cut -d " " -f 1)
PARAMS25_SHASUM=$(echo "$P_CHECKSUMS" | grep "params25" | cut -d " " -f 1)
PARAMS26_SHASUM=$(echo "$P_CHECKSUMS" | grep "params26" | cut -d " " -f 1)

# v0.12.0
VK_CHUNK_SHASUM_HI=$(echo "$R_CHECKSUMS_HI" | grep "vk_chunk.vkey" | cut -d " " -f 1)
VK_BATCH_SHASUM_HI=$(echo "$R_CHECKSUMS_HI" | grep "vk_batch.vkey" | cut -d " " -f 1)
VK_BUNDLE_SHASUM_HI=$(echo "$R_CHECKSUMS_HI" | grep "vk_bundle.vkey" | cut -d " " -f 1)
VRFR_SHASUM_HI=$(echo "$R_CHECKSUMS_HI" | grep "evm_verifier.bin" | cut -d " " -f 1)
CFG2_SHASUM_HI=$(echo "$R_CHECKSUMS_HI" | grep "layer2.config" | cut -d " " -f 1)
CFG4_SHASUM_HI=$(echo "$R_CHECKSUMS_HI" | grep "layer4.config" | cut -d " " -f 1)

# v0.11.4
VK_CHUNK_SHASUM_LO=$(echo "$R_CHECKSUMS_LO" | grep "chunk_vk.vkey" | cut -d " " -f 1)
VK_BATCH_SHASUM_LO=$(echo "$R_CHECKSUMS_LO" | grep "agg_vk.vkey" | cut -d " " -f 1)
VRFR_SHASUM_LO=$(echo "$R_CHECKSUMS_LO" | grep "evm_verifier.bin" | cut -d " " -f 1)
CFG2_SHASUM_LO=$(echo "$R_CHECKSUMS_LO" | grep "layer2.config" | cut -d " " -f 1)
CFG4_SHASUM_LO=$(echo "$R_CHECKSUMS_LO" | grep "layer4.config" | cut -d " " -f 1)

check_shasum() {
	SHASUM=$(shasum -a 256 $1 | cut -d " " -f 1)
	if [ "$SHASUM" != "$2" ]; then
		echo "Shasum mismatch: expected=$2, actual=$SHASUM"
		return 1
	else
		return 0
	fi
}

# check existing file checksums
check_file() {
	if [ -f $1 ]; then
		if ! check_shasum $1 $2; then
			echo "Removing incorrect file $1"
			rm $1
		fi
	fi
}

# check existing common file
check_file "/verifier/params/params20" "$PARAMS20_SHASUM"
check_file "/verifier/params/params21" "$PARAMS21_SHASUM"
check_file "/verifier/params/params24" "$PARAMS24_SHASUM"
check_file "/verifier/params/params25" "$PARAMS25_SHASUM"
check_file "/verifier/params/params26" "$PARAMS26_SHASUM"
# check existing vk_hi file v0.12.0
check_file "/verifier/assets/hi/vk_chunk.vkey" "$VK_CHUNK_SHASUM_HI"
check_file "/verifier/assets/hi/vk_batch.vkey" "$VK_BATCH_SHASUM_HI"
check_file "/verifier/assets/hi/vk_bundle.vkey" "$VK_BUNDLE_SHASUM_HI"
check_file "/verifier/assets/hi/evm_verifier.bin" "$VRFR_SHASUM_HI"
check_file "/verifier/assets/hi/layer2.config" "$CFG2_SHASUM_HI"
check_file "/verifier/assets/hi/layer4.config" "$CFG4_SHASUM_HI"
# check existing vk_lo file v0.11.4
check_file "/verifier/assets/lo/chunk_vk.vkey" "$VK_CHUNK_SHASUM_LO"
check_file "/verifier/assets/lo/agg_vk.vkey" "$VK_BATCH_SHASUM_LO"
check_file "/verifier/assets/lo/evm_verifier.bin" "$VRFR_SHASUM_LO"
check_file "/verifier/assets/lo/layer2.config" "$CFG2_SHASUM_LO"
check_file "/verifier/assets/lo/layer4.config" "$CFG4_SHASUM_LO"

# download missing files
download_file() {
	if [ ! -f $1 ]; then
		mkdir -p $(dirname $1)
		echo "Downloading $1..."
		wget --progress=dot:mega $2 -O $1
		echo "Download completed $1"
		if ! check_shasum $1 $3; then exit 1; fi
	fi
}

# download common
download_file "/verifier/params/params20" "https://circuit-release.s3.us-west-2.amazonaws.com/setup/params20" "$PARAMS20_SHASUM"
download_file "/verifier/params/params21" "https://circuit-release.s3.us-west-2.amazonaws.com/setup/params21" "$PARAMS21_SHASUM"
download_file "/verifier/params/params24" "https://circuit-release.s3.us-west-2.amazonaws.com/setup/params24" "$PARAMS24_SHASUM"
download_file "/verifier/params/params25" "https://circuit-release.s3.us-west-2.amazonaws.com/setup/params25" "$PARAMS25_SHASUM"
download_file "/verifier/params/params26" "https://circuit-release.s3.us-west-2.amazonaws.com/setup/params26" "$PARAMS26_SHASUM"
# download hi v0.12.0
download_file "/verifier/assets/hi/vk_chunk.vkey" "https://circuit-release.s3.us-west-2.amazonaws.com/release-$RELEASE_VERSION_HI/vk_chunk.vkey" "$VK_CHUNK_SHASUM_HI"
download_file "/verifier/assets/hi/vk_batch.vkey" "https://circuit-release.s3.us-west-2.amazonaws.com/release-$RELEASE_VERSION_HI/vk_batch.vkey" "$VK_BATCH_SHASUM_HI"
download_file "/verifier/assets/hi/vk_bundle.vkey" "https://circuit-release.s3.us-west-2.amazonaws.com/release-$RELEASE_VERSION_HI/vk_bundle.vkey" "$VK_BUNDLE_SHASUM_HI"
download_file "/verifier/assets/hi/evm_verifier.bin" "https://circuit-release.s3.us-west-2.amazonaws.com/release-$RELEASE_VERSION_HI/evm_verifier.bin" "$VRFR_SHASUM_HI"
download_file "/verifier/assets/hi/layer2.config" "https://circuit-release.s3.us-west-2.amazonaws.com/release-$RELEASE_VERSION_HI/layer2.config" "$CFG2_SHASUM_HI"
download_file "/verifier/assets/hi/layer4.config" "https://circuit-release.s3.us-west-2.amazonaws.com/release-$RELEASE_VERSION_HI/layer4.config" "$CFG4_SHASUM_HI"
# download low v0.11.4
download_file "/verifier/assets/lo/chunk_vk.vkey" "https://circuit-release.s3.us-west-2.amazonaws.com/release-$RELEASE_VERSION_LO/chunk_vk.vkey" "$VK_CHUNK_SHASUM_LO"
download_file "/verifier/assets/lo/agg_vk.vkey" "https://circuit-release.s3.us-west-2.amazonaws.com/release-$RELEASE_VERSION_LO/agg_vk.vkey" "$VK_BATCH_SHASUM_LO"
download_file "/verifier/assets/lo/evm_verifier.bin" "https://circuit-release.s3.us-west-2.amazonaws.com/release-$RELEASE_VERSION_LO/evm_verifier.bin" "$VRFR_SHASUM_LO"
download_file "/verifier/assets/lo/layer2.config" "https://circuit-release.s3.us-west-2.amazonaws.com/release-$RELEASE_VERSION_LO/layer2.config" "$CFG2_SHASUM_LO"
download_file "/verifier/assets/lo/layer4.config" "https://circuit-release.s3.us-west-2.amazonaws.com/release-$RELEASE_VERSION_LO/layer4.config" "$CFG4_SHASUM_LO"