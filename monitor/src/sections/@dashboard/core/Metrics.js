import { useEffect, useState, useCallback } from 'react';
import { Helmet } from 'react-helmet-async';
import { useParams } from 'react-router-dom';
import PropTypes from 'prop-types';
// @mui
import { Grid, Stack, ButtonGroup, Typography, Container } from '@mui/material';
// sections
import { da } from 'date-fns/locale';
import { getNodes, lookupFamilies, lookupNode } from '../../../api/node';

import TypeSelector from './TypeSelector';
import ZoneSelector from './ZoneSelector';
import FamilySelector from './FamilySelector';
import MetricSelector from './MetricSelector';
import Chart from './Chart';
import DurationButton from './DurationButton';

// ----------------------------------------------------------------------

Metrics.propTypes = {
  defaultType: PropTypes.string,
  defaultZone: PropTypes.string,
  defaultFamily: PropTypes.string,
  defaultName: PropTypes.string,
  enableTypeSelector: PropTypes.bool,
  enableZoneSelector: PropTypes.bool,
  enableFamilySelector: PropTypes.bool,
};

export default function Metrics({
  defaultType,
  defaultZone,
  defaultFamily,
  defaultName,
  enableTypeSelector,
  enableZoneSelector,
  enableFamilySelector,
}) {
  let { type: pType, zone: pZone, family: pFamily, name: pName } = useParams();
  pType = pType || defaultType;
  pZone = pZone || defaultZone;
  pFamily = pFamily || defaultFamily;
  pName = pName || defaultName;
  console.log('params: pZone=%s, pFamily=%s, pName=%s', pZone, pFamily, pName);

  const enabledSelectors = [];
  if (enableTypeSelector) enabledSelectors.push('type');
  if (enableZoneSelector) enabledSelectors.push('zone');
  if (enableFamilySelector) enabledSelectors.push('family');

  const [logLevel, setLogLevel] = useState(`info`);
  const [logContent, setLogContent] = useState(`Select ${enabledSelectors.join(', ')} and name to show data`); // or "Loading"

  const log = (level, content) => {
    setLogLevel(level);
    setLogContent(content);
  };

  const logLevelColor = (level) => {
    switch (level) {
      case 'warn':
        return 'chocolate';
      case 'error':
        return 'red';
      default:
        return 'gray';
    }
  };

  // const curveType = "area";
  // const curveFill = "gradient";
  const curveType = 'line';
  const curveFill = 'solid';

  // list all available data items
  const types = ['metrics', 'runtime'];
  const [nodes] = useState(getNodes());
  const [zones, setZones] = useState([]);
  const [families, setFamilies] = useState([]);
  const [metrics, setMetrics] = useState([]);
  const [durations, setDurations] = useState([]);

  // current selected data item
  const [type, setType] = useState(pType || '');
  const [zone, setZone] = useState(pZone || '');
  const [family, setFamily] = useState(pFamily || '');
  const [metric, setMetric] = useState({});
  const [duration, setDuration] = useState({});

  // data for UI
  const [chartType, setChartType] = useState('line');
  const [chartXAxisType, setChartXAxisType] = useState('time');
  const [chartLabels, setChartLabels] = useState([]);
  const [chartData, setChartData] = useState([]);
  const [chartYLabelFormatter, setChartYLabelFormatter] = useState('');

  // curve properties
  const propertyCur = 1;
  const propertySample = 2;
  const propertyInterval = 4;
  const propertyMin = 8;
  const propertyMax = 16;

  const nameOfProperty = (p) => {
    p = parseInt(p, 10);
    switch (p) {
      case propertyCur:
        return 'cur';
      case propertySample:
        return 'sample';
      case propertyInterval:
        return 'interval';
      case propertyMin:
        return 'min';
      case propertyMax:
        return 'max';
      default:
        return `property(${p})`;
    }
  };

  const handleChangeType = (value) => {
    console.log(`type changed to ${value}`);
    setMetric({});
    setType(value);
  };

  const handleChangeZone = (value) => {
    console.log(`zone changed to ${value}`);
    setZone(value);
  };

  const handleChangeFamily = (value) => {
    console.log(`family changed to ${value}`);
    setFamily(value);
  };

  const handleChangeName = (value) => {
    console.log(`name changed to ${value}`);
    for (let i = 0; i < metrics.length; i += 1) {
      if (metrics[i].name === value) {
        setMetric(metrics[i]);
        return;
      }
    }
    setMetric({});
  };

  const handleChangeDuration = (d) => {
    setDuration({ ...d });
  };

  // update data for curve chart
  const updateCurveChartData = useCallback((data) => {
    const minute = 60;
    const hour = 60 * minute;

    data.values = data.values || [];
    data.duration = data.duration || { uint: 0, count: 0 };

    const labels = [];
    const charts = [];
    const formatDigits = (x, n) => x.toLocaleString('en-US', { minimumIntegerDigits: n, useGrouping: false });
    Object.keys(data.values).forEach((p) => {
      const values = data.values[p];
      const chart = {
        type: curveType,
        fill: curveFill,
        name: nameOfProperty(p),
        data: values,
      };
      charts.push(chart);
      if (labels.length === 0) {
        for (let i = 0; i < values.length; i += 1) {
          const t = new Date((data.timestamp - (values.length - i - 1) * data.duration.unit) * 1000);
          const hh = formatDigits(t.getHours(), 2);
          const MM = formatDigits(t.getMinutes(), 2);
          const ss = formatDigits(t.getSeconds(), 2);
          if (data.duration.unit < hour) {
            labels.push(`${hh}:${MM}:${ss}`);
          } else {
            const yyyy = formatDigits(t.getFullYear(), 4);
            const mm = formatDigits(t.getMonth(), 2);
            const dd = formatDigits(t.getDate(), 2);
            labels.push(`${yyyy}/${mm}/${dd} ${hh}:${MM}:${ss}`);
          }
        }
      }
    });
    if (data.duration.unit < hour) {
      setChartXAxisType('time');
    } else {
      setChartXAxisType('datetime');
    }
    setChartType('line');
    setChartLabels(labels);
    setChartData(charts);
    if (data.descriptor) {
      setChartYLabelFormatter(data.descriptor.formatter || '');
    } else {
      setChartYLabelFormatter('');
    }
  }, []);

  // update data for histogram chart
  const updateHistogramChartData = useCallback((data) => {
    const chart = {
      type: 'bar',
      fill: 'gradient',
    };
    const findRange = (seperators, start, end) => {
      if (start === '-Inf') {
        return 0;
      }
      if (end === '+Inf') {
        return seperators.length;
      }

      start = parseFloat(start);
      end = parseFloat(end);
      if (end <= seperators[0]) {
        return 0;
      }
      if (start >= seperators[seperators.length - 1]) {
        return seperators.length;
      }

      let maxOverlap = 0;
      let maxOverlapIndex = -1;
      for (let i = 0; i < seperators.length; i += 1) {
        // overlap between [start, end] and seperators[i,i+1]
        const overlap = Math.min(end, seperators[i + 1]) - Math.max(start, seperators[i]);
        if (overlap > maxOverlap || maxOverlapIndex === -1) {
          maxOverlap = overlap;
          maxOverlapIndex = i;
        }
      }
      if (maxOverlap >= 0) {
        return maxOverlapIndex;
      }

      if (start <= seperators[0]) {
        return 0;
      }
      return seperators.length;
    };
    const mergeValues = (seperators, formatLabel) => {
      const labels = [];
      const values = [];
      labels.length = seperators.length + 1;
      values.length = seperators.length + 1;
      labels[0] = `<${formatLabel(seperators[0])}`;
      for (let i = 1; i < seperators.length; i += 1) {
        labels[i] = `${formatLabel(seperators[i - 1])}~${formatLabel(seperators[i])}`;
      }
      labels[seperators.length] = `>=${formatLabel(seperators[seperators.length - 1])}`;
      values.fill(0);
      for (let i = 1; i < data.labels.length; i += 1) {
        const index = findRange(seperators, data.labels[i - 1], data.labels[i]);
        values[index] += data.values[i - 1];
      }
      return { labels, values };
    };
    if (data.descriptor) {
      // data: data.values,
      if (data.descriptor.formatter === 'histogram:bytes') {
        const k = 1024;
        const m = 1024 * k;
        const g = 1024 * m;
        const formatSpace = (s) => {
          if (s < k) {
            return s;
          }
          if (s < m) {
            return `${s / k}K`;
          }
          if (s < g) {
            return `${s / m}M`;
          }
          return `${s / g}G`;
        };
        const { labels, values } = mergeValues(
          [4, 16, 64, 256, 1 * k, 4 * k, 16 * k, 64 * k, 256 * k, 1 * m, 4 * m, 16 * m, 64 * m, 256 * m, 1024 * m],
          formatSpace
        );
        setChartLabels(labels);
        chart.data = values;
      } else if (data.descriptor.formatter === 'histogram:seconds') {
        const ms = 0.001;
        const s = 1000 * ms;
        const formatDuration = (d) => {
          if (d < s) {
            return `${Math.floor(d / ms + 0.5)}ms`;
          }
          return `${Math.floor(d / s + 0.5)}s`;
        };
        const { labels, values } = mergeValues(
          [
            1 * ms,
            2 * ms,
            5 * ms,
            10 * ms,
            20 * ms,
            50 * ms,
            100 * ms,
            200 * ms,
            500 * ms,
            1 * s,
            2 * s,
            5 * s,
            10 * s,
          ],
          formatDuration
        );
        setChartLabels(labels);
        chart.data = values;
      }
    }
    if (!chart.data) {
      chart.data = data.values;
      const labels = [];
      for (let i = 0; i < data.labels.length; i += 1) {
        labels.push(data.labels[i]);
      }
      setChartLabels(labels);
    }
    setChartXAxisType('category');
    setChartType('bar');
    setChartData([chart]);
    if (data.descriptor) {
      setChartYLabelFormatter(data.descriptor.formatter || '');
    } else {
      setChartYLabelFormatter('');
    }
  }, []);

  // update data for chart
  const updateChartData = useCallback(
    (data) => {
      console.log('set chart from data', data);
      if (data.kind === 'curve') {
        updateCurveChartData(data);
        return;
      }
      if (data.kind === 'histogram') {
        updateHistogramChartData(data);
        return;
      }
      log('error', `data kind should be one of (curve,histogram), but got ${data.kind}`);
    },
    [updateCurveChartData, updateHistogramChartData]
  );

  // refreshes zones by nodes
  useEffect(() => {
    const zones = [];
    for (let i = 0; i < nodes.length; i += 1) {
      zones.push(nodes[i].zone);
    }
    zones.sort();
    for (let i = zones.length - 1; i > 0; i -= 1) {
      if (zones[i] === zones[i - 1]) {
        zones.splice(i, 1);
      }
    }
    setZones(zones);
    console.log('refreshes zones by nodes', zones);
  }, [nodes]);

  // refreshes families for current zone
  useEffect(() => {
    setFamilies([]);
    if (!zone) {
      return;
    }
    const families = lookupFamilies(zone);
    families.sort();
    for (let i = families.length - 1; i > 0; i -= 1) {
      if (families[i] === families[i - 1]) {
        families.splice(i, 1);
      }
    }
    setFamilies(families);
    console.log('refreshes families by for current zone', families);
  }, [zone]);

  // refreshes metrics for current zone and family
  useEffect(() => {
    setMetrics([]);
    setMetric({});
    if (!zone || !family) {
      return;
    }
    const node = lookupNode(zone, family);
    if (!node) {
      log('error', `Node not found by zone ${zone} and family ${family}`);
      return;
    }
    const url = `${node.address}/${type}/list`;
    log('info', `Loading ${url}...`);
    fetch(url)
      .then((response) => response.json())
      .then((data) => {
        if (data.error) {
          log('warn', `An error returned from ${url}: ${data.error}`);
          return;
        }
        data.metrics.sort((a, b) => {
          if (a.descriptor && b.descriptor) {
            if (a.descriptor.level < b.descriptor.level) {
              return -1;
            }
            if (a.descriptor.level > b.descriptor.level) {
              return 1;
            }
          }
          return a.name < b.name ? -1 : a.name > b.name ? 1 : 0;
        });
        setMetrics(data.metrics);
        console.log('refreshes metrics for current zone and family', data.metrics);
        log('info', 'Select name to show chart');
      })
      .catch((reason) => {
        log('error', `Failed to fetch ${url}: ${reason}`);
      });
  }, [type, zone, family]);

  // refreshes durations/duration for current zone, family and name
  useEffect(() => {
    if (!zone || !family || !metric || !metric.name) {
      return;
    }
    const node = lookupNode(zone, family);
    if (!node) {
      log('error', `Node not found by zone ${zone} and family ${family}`);
    }
    const values = [];
    for (let i = 0; i < metrics.length; i += 1) {
      if (metrics[i].name === metric.name && metrics[i].durations && metrics[i].durations.length > 0) {
        for (let j = 0; j < metrics[i].durations.length; j += 1) {
          values.push({ ...metrics[i].durations[j] });
        }
        break;
      }
    }
    console.log('refreshes durations/duration for current zone, family and name', values);
    setDurations(values);
    let found = false;
    if (duration && duration.unit > 0) {
      for (let i = 0; i < values.length; i += 1) {
        if (values[i].unit === duration.unit && values[i].count === duration.count) {
          found = true;
          break;
        }
      }
    }
    if (!found) {
      if (values.length === 0 && (!duration || !duration.unit)) {
        return;
      }
      setDuration(values.length > 0 ? { ...values[0] } : { uint: 0, count: 0 });
    }
  }, [metrics, zone, family, metric, duration]);

  // refreshes chart data for current zone, family, metric and duration
  useEffect(() => {
    setChartData([]);
    if (!zone || !family || !metric || !metric.name) {
      return;
    }
    if (metric.durations && metric.durations.length > 0 && (!duration || !duration.unit)) {
      return;
    }
    console.log('refreshes chart data for current zone, family, name and druation');
    const node = lookupNode(zone, family);
    if (!node) {
      log('error', `Node not found by zone ${zone} and family ${family}`);
      return;
    }
    const url = `${node.address}/${type}/get?name=${encodeURIComponent(metric.name)}&duration.unit=${
      (duration && duration.unit) || 0
    }&duration.count=${(duration && duration.count) || 0}`;
    log('info', `Loading ${url}...`);
    fetch(url)
      .then((response) => response.json())
      .then((data) => {
        if (data.error) {
          log('warn', `An error returned from ${data.url}: ${data.error}`);
          return;
        }
        updateChartData(data);
      })
      .catch((reason) => {
        log('error', `Failed to fetch ${url}: ${reason}`);
      });
  }, [type, zone, family, metric, duration, updateChartData]);

  const nameOfMetric = (metric) => {
    if (metric && metric.name) {
      return metric.name;
    }
    return '';
  };

  const descriptionOfMetric = (metric) => {
    if (metric && metric.descriptor && metric.descriptor.description) {
      return `${metric.descriptor.description}`;
    }
    return '';
  };

  return (
    <>
      <Helmet>
        <title> {`${type} ${zone || ''} ${family || ''} ${(metric && metric.name) || ''}`} </title>
      </Helmet>

      <Container maxWidth="xl">
        <Typography variant="h3" paragraph>
          {type}
        </Typography>

        {/* zone, family and name selector */}
        <Stack direction="row" alignItems="center" spacing={2}>
          {enableTypeSelector && (
            <TypeSelector type={type} items={types} defaultValue={pType} handleChange={handleChangeType} />
          )}
          {enableZoneSelector && (
            <ZoneSelector type={type} items={zones} defaultValue={pZone} handleChange={handleChangeZone} />
          )}
          {enableFamilySelector && (
            <FamilySelector type={type} items={families} defaultValue={pFamily} handleChange={handleChangeFamily} />
          )}
          <MetricSelector type={type} items={metrics} defaultValue={pName} handleChange={handleChangeName} />
        </Stack>

        {/* duration selector */}
        {chartData.length > 0 && metric && durations && durations.length > 0 && (
          <Stack direction="row" alignItems="center" spacing={2}>
            <ButtonGroup variant="outlined" sx={{ paddingTop: '24px' }}>
              {durations.map((item) => (
                <DurationButton
                  key={`${item.unit}_${item.count}`}
                  duration={item}
                  selected={duration && item.unit === duration.unit && item.count === duration.count}
                  onClick={handleChangeDuration}
                />
              ))}
            </ButtonGroup>
          </Stack>
        )}

        {/* show chart */}
        <Grid container spacing={3} sx={{ paddingTop: '24px' }}>
          <Grid item xs={12} md={12} lg={12}>
            {chartData.length > 0 ? (
              // show chart
              <Chart
                title={`${nameOfMetric(metric)}`}
                subheader={`${descriptionOfMetric(metric)}`}
                type={chartType}
                labels={chartLabels}
                data={chartData}
                xAxisType={chartXAxisType}
                yLabelFormatter={chartYLabelFormatter}
              />
            ) : (
              // or show tips
              <Typography
                align="center"
                sx={{
                  paddingTop: '48px',
                  color: `${logLevelColor(logLevel)}`,
                }}
              >
                {logContent}
              </Typography>
            )}
          </Grid>
        </Grid>
      </Container>
    </>
  );
}
