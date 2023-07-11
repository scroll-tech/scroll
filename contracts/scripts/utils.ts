import * as fs from "fs";
import * as path from "path";
import editJsonFile from "edit-json-file";

const CONFIG_FILE_DIR = path.join(__dirname, "../", "deployments");

export function selectAddressFile(network: string) {
  if (!fs.existsSync(CONFIG_FILE_DIR)) {
    fs.mkdirSync(CONFIG_FILE_DIR, { recursive: true });
  }

  let filename: string;
  if (["hardhat", "l1geth", "l2geth"].includes(network)) {
    filename = path.join(CONFIG_FILE_DIR, `${network}.json`);
  } else {
    throw new Error(`network ${network} not supported yet`);
  }

  const addressFile = editJsonFile(filename, {
    stringify_eol: true,
    autosave: true,
  });

  return addressFile;
}
