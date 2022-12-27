import * as React from 'react';
import PropTypes from 'prop-types';

import { FormControl, Autocomplete, TextField } from '@mui/material';
import StarIcon from '@mui/icons-material/Star';
import StarOutline from '@mui/icons-material/StarOutline';

MetricSelector.propTypes = {
  type: PropTypes.string,
  items: PropTypes.arrayOf(PropTypes.string),
  defaultValue: PropTypes.string,
  handleChange: PropTypes.func,
};

export default function MetricSelector({ type, items, defaultValue, handleChange }) {
  const id = `${type}-select-family`;

  const [metric, setMetric] = React.useState(items.find((item) => item.name === defaultValue) || {});
  const [input, setInput] = React.useState('');

  const onChange = (event, value) => {
    setMetric(value);
    handleChange((value && value.name) || '');
  };

  const onInputChange = (event, value, reason) => {
    setInput(value);
  };

  const getIndicesOf = (searchStr, str, caseSensitive) => {
    const searchStrLen = searchStr.length;
    if (searchStrLen === 0) {
      return [];
    }
    let startIndex = 0;
    const indices = [];
    if (!caseSensitive) {
      str = str.toLowerCase();
      searchStr = searchStr.toLowerCase();
    }
    let index = 0;
    do {
      index = str.indexOf(searchStr, startIndex);
      if (index < 0) {
        break;
      }
      indices.push(index);
      startIndex = index + searchStrLen;
    } while (index > -1);
    return indices;
  };

  const IMPORTANT = 0;
  const INFO = 1;

  const renderLevel = (level) => {
    const fontSize = 12;
    switch (level) {
      case IMPORTANT:
        return <StarIcon sx={{ fontSize }} />;
      case INFO:
        return <StarOutline sx={{ fontSize }} />;
      default:
        return <StarOutline sx={{ fontSize }} color="disabled" />;
    }
  };

  const renderItem = (item) => {
    const indices = getIndicesOf(input, item.name);
    const elements = [];
    const level = (item.descriptor && item.descriptor.level) || IMPORTANT;
    elements.push(renderLevel(level));
    if (indices.length === 0) {
      elements.push(item.name);
    } else {
      let prev = 0;
      for (let i = 0; i < indices.length; i += 1) {
        elements.push(item.name.substring(prev, indices[i]));
        elements.push(<strong>{input}</strong>);
        prev = indices[i] + input.length;
      }
      elements.push(item.name.substring(indices[indices.length - 1] + input.length));
    }
    return elements;
  };

  return (
    <FormControl fullWidth>
      <Autocomplete
        value={metric}
        id={id}
        options={items}
        getOptionLabel={(item) => (item && item.name) || ''}
        renderOption={(props, item) => <li {...props}>{renderItem(item)}</li>}
        onChange={onChange}
        renderInput={(params) => <TextField {...params} label="Name" />}
        onInputChange={onInputChange}
      />
    </FormControl>
  );
}
