// sections
import { Metrics } from '../sections/@dashboard/core';

// ----------------------------------------------------------------------

export default function RuntimePage() {
  return <Metrics defaultType="runtime" enableZoneSelector enableFamilySelector />;
}
