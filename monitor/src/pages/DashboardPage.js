import { Helmet } from 'react-helmet-async';
import { filter } from 'lodash';
import { sentenceCase } from 'change-case';
import { useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
// @mui
import {
  Card,
  Table,
  Stack,
  Paper,
  Button,
  TableRow,
  TableBody,
  TableCell,
  Container,
  Typography,
  TableContainer,
} from '@mui/material';
// components
import Label from '../components/label';
import Scrollbar from '../components/scrollbar';
// sections
import { NodeListHead, NodeListToolbar } from '../sections/@dashboard/app';
// mock
import { getNodes } from '../api/node';

// ----------------------------------------------------------------------

const TABLE_HEAD = [
  { id: 'zone', label: 'Zone', alignRight: false },
  { id: 'family', label: 'Family', alignRight: false },
  { id: 'address', label: 'Address', alignRight: false },
  { id: 'status', label: 'Status', alignRight: false },
  { id: 'infos', label: 'Informations', alignRight: false },
  { id: '' },
];

// ----------------------------------------------------------------------

function descendingComparator(a, b, orderBy) {
  if (b[orderBy] < a[orderBy]) {
    return -1;
  }
  if (b[orderBy] > a[orderBy]) {
    return 1;
  }
  return 0;
}

function getComparator(order, orderBy) {
  return order === 'desc'
    ? (a, b) => descendingComparator(a, b, orderBy)
    : (a, b) => -descendingComparator(a, b, orderBy);
}

function applySortFilter(array, comparator, query) {
  const stabilizedThis = array.map((el, index) => [el, index]);
  stabilizedThis.sort((a, b) => {
    const order = comparator(a[0], b[0]);
    if (order !== 0) return order;
    return a[1] - b[1];
  });
  if (query) {
    return filter(
      array,
      (node) =>
        node.zone.toLowerCase().indexOf(query.toLowerCase()) !== -1 ||
        node.family.toLowerCase().indexOf(query.toLowerCase()) !== -1
    );
  }
  return stabilizedThis.map((el) => el[0]);
}

export default function DashboardPage() {
  const NODELIST = getNodes();
  const [order, setOrder] = useState('asc');
  const [orderBy, setOrderBy] = useState('zone');
  const [filterName, setFilterName] = useState('');

  const handleRequestSort = (event, property) => {
    const isAsc = orderBy === property && order === 'asc';
    setOrder(isAsc ? 'desc' : 'asc');
    setOrderBy(property);
  };

  const handleFilterByName = (event) => {
    setFilterName(event.target.value);
  };

  const filteredNodes = applySortFilter(NODELIST, getComparator(order, orderBy), filterName);

  const isNotFound = !filteredNodes.length && !!filterName;

  const statusColor = (status) => {
    switch (status) {
      case 'running':
        return 'success';
      case 'closed':
        return 'error';
      default:
        return 'default';
    }
  };

  return (
    <>
      <Helmet>
        <title> Dashboard </title>
      </Helmet>

      <Container>
        <Card>
          <NodeListToolbar filterName={filterName} onFilterName={handleFilterByName} />

          <Scrollbar>
            <TableContainer sx={{ minWidth: 800 }}>
              <Table>
                <NodeListHead
                  order={order}
                  orderBy={orderBy}
                  headLabel={TABLE_HEAD}
                  rowCount={NODELIST.length}
                  onRequestSort={handleRequestSort}
                />
                <TableBody>
                  {filteredNodes.map((row) => {
                    const { address, family, status, zone } = row;

                    return (
                      <TableRow hover key={address} tabIndex={-1}>
                        <TableCell align="left">{zone}</TableCell>
                        <TableCell align="left">{family}</TableCell>
                        <TableCell align="left">{address}</TableCell>
                        <TableCell align="left">
                          <Label color={statusColor(status)}>{sentenceCase(status)}</Label>
                        </TableCell>

                        <TableCell align="left">
                          <Stack direction="row" alignItems="center" spacing={2}>
                            <Button
                              variant="outlined"
                              component={RouterLink}
                              to={`/dashboard/runtime/${zone}/${family}`}
                            >
                              Runtime
                            </Button>
                            <Button
                              variant="outlined"
                              component={RouterLink}
                              to={`/dashboard/metrics/${zone}/${family}`}
                            >
                              Metrics
                            </Button>
                          </Stack>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                  {/*
                  emptyRows > 0 && (
                    <TableRow style={{ height: 53 * emptyRows }}>
                      <TableCell colSpan={6} />
                    </TableRow>
                  )
                  */}
                </TableBody>

                {isNotFound && (
                  <TableBody>
                    <TableRow>
                      <TableCell align="center" colSpan={6} sx={{ py: 3 }}>
                        <Paper
                          sx={{
                            textAlign: 'center',
                          }}
                        >
                          <Typography variant="h6" paragraph>
                            Not found
                          </Typography>

                          <Typography variant="body2">
                            No results found for &nbsp;
                            <strong>&quot;{filterName}&quot;</strong>.
                            <br /> Try checking for typos or using complete words.
                          </Typography>
                        </Paper>
                      </TableCell>
                    </TableRow>
                  </TableBody>
                )}
              </Table>
            </TableContainer>
          </Scrollbar>

          {/*
          <TablePagination
            rowsPerPageOptions={[5, 10, 25]}
            component="div"
            count={NODELIST.length}
            rowsPerPage={rowsPerPage}
            page={page}
            onPageChange={handleChangePage}
            onRowsPerPageChange={handleChangeRowsPerPage}
          />
          */}
        </Card>
      </Container>
    </>
  );
}
