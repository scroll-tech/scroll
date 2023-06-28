export interface DomainDeployment {
  ForwarderAddress: string;

  ScrollSafeAddress: string;
  ScrollTimelockAddress: string;

  CouncilSafeAddress: string;
  CouncilTimelockAddress: string;
}

export interface Deployment {
  L1: DomainDeployment;
  L2: DomainDeployment;
}

export interface Config {
  [key: string]: Deployment;
}

const config: Config = {
  testnet: {
    L1: {
      ForwarderAddress: "0x0000000000000000000000000000000000000000",
      ScrollSafeAddress: "0x0000000000000000000000000000000000000000",
      ScrollTimelockAddress: "0x0000000000000000000000000000000000000000",
      CouncilSafeAddress: "0x0000000000000000000000000000000000000000",
      CouncilTimelockAddress: "0x0000000000000000000000000000000000000000",
    },
    L2: {
      ForwarderAddress: "0xA51c1fc2f0D1a1b8494Ed1FE312d7C3a78Ed91C0",
      ScrollSafeAddress: "0xa513E6E4b8f2a923D98304ec87F64353C4D5C853",
      ScrollTimelockAddress: "0x8A791620dd6260079BF849Dc5567aDC3F2FdC318",
      CouncilSafeAddress: "0x0000000000000000000000000000000000000000",
      CouncilTimelockAddress: "0x0000000000000000000000000000000000000000",
    },
  },
};

export const getConfig = (network: string, domain: string): DomainDeployment => {
  if (network in config) {
    if (domain in config[network]) {
      return config[network][domain as keyof Deployment];
    } else {
      throw new Error(`Invalid domain: ${domain}`);
    }
  } else {
    throw new Error(`Invalid network: ${network}`);
  }
};
