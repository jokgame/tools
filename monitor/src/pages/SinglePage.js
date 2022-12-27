// sections
import { Metrics } from '../sections/@dashboard/core';

// ----------------------------------------------------------------------

export default function SinglePage() {
  return <Metrics defaultType="metrics" defaultZone="local" defaultFamily="local" enableTypeSelector />;
}
