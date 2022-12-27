// ----------------------------------------------------------------------

const nodes = [
  {
    zone: 'local',
    family: 'local',
    address: '/gauge/api',
    status: 'success',
  },
];

const indices = (() => {
  const mapping = {};
  nodes.forEach((x) => {
    mapping[x.address] = x;
  });
  return mapping;
})();

function getNodes() {
  return nodes;
}

function getNode(address) {
  const index = indices[address];
  if (index || index === 0) {
    return nodes[index];
  }
  return null;
}

function lookupNode(zone, family) {
  for (let i = 0; i < nodes.length; i += 1) {
    if (nodes[i].zone === zone && nodes[i].family === family) {
      return nodes[i];
    }
  }
  return null;
}

function lookupFamilies(zone) {
  const families = [];
  for (let i = 0; i < nodes.length; i += 1) {
    if (nodes[i].zone === zone) {
      families.push(nodes[i].family);
    }
  }
  return families;
}

export { getNodes, getNode, lookupNode, lookupFamilies };
