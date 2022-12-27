// component
import SvgColor from '../../../components/svg-color';

// ----------------------------------------------------------------------

const icon = (name) => <SvgColor src={`/assets/icons/navbar/${name}.png`} sx={{ width: 1, height: 1 }} />;

const navConfig = [
  {
    title: 'dashboard',
    path: '/dashboard/app',
    icon: icon('ic_dashboard'),
  },
  {
    title: 'runtime',
    path: '/dashboard/runtime',
    icon: icon('ic_runtime'),
  },
  {
    title: 'metrics',
    path: '/dashboard/metrics',
    icon: icon('ic_metrics'),
  },
];

export default navConfig;
