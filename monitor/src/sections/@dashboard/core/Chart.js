import PropTypes from 'prop-types';
import ReactApexChart from 'react-apexcharts';
// @mui
import { Card, CardHeader, Box } from '@mui/material';
// components
import { useChart } from '../../../components/chart';

// ----------------------------------------------------------------------

Chart.propTypes = {
  title: PropTypes.string,
  subheader: PropTypes.string,
  type: PropTypes.string,
  xAxisType: PropTypes.string.isRequired,
  data: PropTypes.array.isRequired,
  labels: PropTypes.arrayOf(PropTypes.string).isRequired,
  yLabelFormatter: PropTypes.string,
};

export default function Chart({ title, subheader, type, xAxisType, labels, yLabelFormatter, data, ...other }) {
  const KB = 1024;
  const MB = 1024 * KB;
  const GB = 1024 * MB;
  const TB = 1024 * GB;
  const PB = 1024 * TB;

  const formatNumberWithUnit = (value, unit, unitName) => {
    value = (value - (value % unit)) / unit;
    if (Number.isNaN(value)) {
      return value;
    }
    return `${value}${unitName}`;
  };

  const humanReadableBytes = (value) => {
    if (Number.isNaN(value)) {
      return '';
    }
    if (value < 10 * KB) {
      return value;
    }
    if (value < 10 * MB) {
      return formatNumberWithUnit(value, KB, 'K');
    }
    if (value < 10 * GB) {
      return formatNumberWithUnit(value, MB, 'M');
    }
    if (value < 10 * TB) {
      return formatNumberWithUnit(value, GB, 'G');
    }
    if (value < 10 * PB) {
      return formatNumberWithUnit(value, TB, 'T');
    }
    if (value > 0) {
      return formatNumberWithUnit(value, PB, 'P');
    }
    return value;
  };

  const percent = (value) => `${value}%`;

  const yLabelFormat = (value) => {
    switch (yLabelFormatter) {
      case 'bytes:human-readable':
        return humanReadableBytes(value);
      case 'percent':
        return percent(value);
      default:
        if (Number.isInteger(value)) {
          return humanReadableBytes(value);
        }
        return value;
    }
  };
  const chartOptions = useChart({
    plotOptions: { bar: { columnWidth: '16%' } },
    fill: { type: data.map((i) => i.fill) },
    labels,
    xaxis: { type: { xaxisType: xAxisType } },
    yaxis: { labels: { formatter: yLabelFormat } },
    tooltip: {
      shared: true,
      intersect: false,
      y: {
        formatter: (y) => {
          if (typeof y !== 'undefined') {
            return `${y.toFixed(0)}`;
          }
          return y;
        },
      },
    },
  });

  return (
    <Card {...other}>
      <CardHeader title={title} subheader={subheader} />

      <Box sx={{ p: 3, pb: 1 }} dir="ltr">
        <ReactApexChart type={type || 'line'} series={data} options={chartOptions} height={364} />
      </Box>
    </Card>
  );
}
