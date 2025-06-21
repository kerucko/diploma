import json
import logging
from typing import Dict, Any, Optional, Tuple
import uuid
import random
import math

from confluent_kafka import Consumer, Producer, KafkaError, KafkaException
import pyomo.environ as pyo
from pyomo.opt import SolverFactory, SolverStatus, TerminationCondition

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger('kafka_pyomo_service')

KAFKA_CONFIG = {
    'bootstrap.servers': 'localhost:29092',
    'group.id': 'pyomo_solver_group',
    'auto.offset.reset': 'earliest'
}

SOLVER_NAME = 'glpk'  

class TaskType:
    MILP = 'milp'  # Mixed-Integer Linear Programming
    LP = 'lp'      # Linear Programming
    
class KafkaPyomoService:
    def __init__(self, kafka_config: Dict[str, str], 
                 input_topic: str = 'tasks', 
                 output_topic: str = 'completed',
                 solver_name: str = SOLVER_NAME):

        self.kafka_config = kafka_config
        self.input_topic = input_topic
        self.output_topic = output_topic
        self.solver_name = solver_name
        
        self.consumer = Consumer(kafka_config)
        self.producer = Producer(kafka_config)
        
        if not SolverFactory(solver_name).available():
            logger.error(f"Солвер {solver_name} недоступен!")
            raise ValueError(f"Солвер {solver_name} недоступен!")
        
        self.supported_solvers = [SOLVER_NAME, 'sa']

            
    def start(self):
        try:
            self.consumer.subscribe([self.input_topic])
            logger.info(f"Сервис запущен и прослушивает топик '{self.input_topic}'")
            
            while True:
                msg = self.consumer.poll(timeout=1.0)
                
                if msg is None:
                    continue
                    
                if msg.error():
                    if msg.error().code() == KafkaError._PARTITION_EOF:
                        logger.warning(f"Достигнут конец раздела: {msg.error()}")
                    else:
                        logger.error(f"Ошибка Kafka: {msg.error()}")
                    continue
                
                try:
                    task_data = json.loads(msg.value().decode('utf-8'))
                    logger.info(f"Получена задача: {task_data.get('id')}")
                    
                    if not all(k in task_data for k in ['id', 'type', 'task']):
                        logger.error(f"Неверный формат сообщения: {task_data}")
                        continue
                        
                    solution = self._solve_task(task_data)
                    
                    self._send_solution(task_data['id'], solution)
                    
                except json.JSONDecodeError:
                    logger.error(f"Ошибка декодирования JSON: {msg.value()}")
                except Exception as e:
                    logger.error(f"Ошибка при обработке задачи: {e}")
                    
        except KeyboardInterrupt:
            logger.info("Сервис остановлен пользователем")
        finally:
            self.consumer.close()
            
    def _solve_task(self, task_data: Dict[str, Any]) -> Dict[str, Any]:
        task_id = task_data['id']
        task_type = task_data['type']
        task = task_data['task']
        
        logger.info(f"Решение задачи {task_id} типа {task_type}")
        
        if task_type not in [TaskType.MILP, TaskType.LP]:
            return {
                'status': 'error',
                'message': f'Неподдерживаемый тип задачи: {task_type}'
            }
        
        solver_type = task_data.get('solver', self.solver_name)
        if solver_type not in self.supported_solvers:
            return {'status': 'error', 'message': f"Солвер '{solver_type}' не поддерживается"}

        if solver_type == 'sa':
            return self._solve_with_sa(task_data['task'])

                
        try:
            model, error = self._create_model_from_json(task, task_type)
            
            if error:
                return {
                    'status': 'error',
                    'message': error
                }
                
            solver = SolverFactory(self.solver_name)
            result = solver.solve(model, tee=False)
            
            if (result.solver.status == SolverStatus.ok and 
                result.solver.termination_condition == TerminationCondition.optimal):
                
                variables = {}
                for v in model.component_objects(pyo.Var, active=True):
                    variables[v.name] = {
                        str(index): pyo.value(v[index]) 
                        for index in v
                    }
                
                objective_value = pyo.value(model.objective)
                
                return {
                    'status': 'success',
                    'objective_value': objective_value,
                    'variables': variables,
                    'solver_status': str(result.solver.status),
                    'termination_condition': str(result.solver.termination_condition)
                }
                
            elif result.solver.termination_condition == TerminationCondition.infeasible:
                return {
                    'status': 'infeasible',
                    'message': 'Задача не имеет допустимых решений',
                    'solver_status': str(result.solver.status),
                    'termination_condition': str(result.solver.termination_condition)
                }
            else:
                return {
                    'status': 'error',
                    'message': f'Проблема с решением: {result.solver.status}, {result.solver.termination_condition}',
                    'solver_status': str(result.solver.status),
                    'termination_condition': str(result.solver.termination_condition)
                }
                
        except Exception as e:
            logger.error(f"Ошибка при решении задачи {task_id}: {e}")
            return {
                'status': 'error',
                'message': str(e)
            }
    
    def _create_model_from_json(self, task_json: Dict[str, Any], task_type: str) -> Tuple[Optional[pyo.ConcreteModel], Optional[str]]:
        try:
            required_fields = ['variables', 'objective', 'constraints']
            if not all(field in task_json for field in required_fields):
                missing = [f for f in required_fields if f not in task_json]
                return None, f"Отсутствуют обязательные поля: {', '.join(missing)}"
                
            model = pyo.ConcreteModel()
            
            for var_name, var_data in task_json['variables'].items():
                domain = pyo.Reals
                if 'domain' in var_data:
                    if var_data['domain'] == 'binary':
                        domain = pyo.Binary
                    elif var_data['domain'] == 'integer':
                        domain = pyo.Integers
                
                bounds = (None, None)
                if 'lb' in var_data:
                    bounds = (var_data['lb'], bounds[1])
                if 'ub' in var_data:
                    bounds = (bounds[0], var_data['ub'])
                
                setattr(model, var_name, pyo.Var(domain=domain, bounds=bounds))
            
            obj_expr = self._parse_expression(model, task_json['objective']['expression'])
            sense = pyo.minimize if task_json['objective'].get('sense', 'min') == 'min' else pyo.maximize
            model.objective = pyo.Objective(expr=obj_expr, sense=sense)
            
            model.constraints = pyo.ConstraintList()
            for i, constraint in enumerate(task_json['constraints']):
                lhs_expr = self._parse_expression(model, constraint['lhs'])
                rhs_expr = self._parse_expression(model, constraint['rhs'])
                
                if constraint['sense'] == '==':
                    model.constraints.add(lhs_expr == rhs_expr)
                elif constraint['sense'] == '<=':
                    model.constraints.add(lhs_expr <= rhs_expr)
                elif constraint['sense'] == '>=':
                    model.constraints.add(lhs_expr >= rhs_expr)
                else:
                    return None, f"Неподдерживаемый оператор ограничения: {constraint['sense']}"
            
            return model, None
            
        except Exception as e:
            logger.error(f"Ошибка при создании модели: {e}")
            return None, str(e)

    def _solve_with_sa(self, task_json: Dict[str, Any]) -> Dict[str, Any]:
        try:
            vars_info = task_json['variables']
            objective_expr = task_json['objective']['expression']
            constraints = task_json['constraints']
            sense = task_json['objective'].get('sense', 'min')

            def evaluate(assignment):
                model_mock = type('MockModel', (), assignment)
                return self._parse_expression(model_mock, objective_expr)

            def is_feasible(assignment):
                model_mock = type('MockModel', (), assignment)
                for cons in constraints:
                    lhs = self._parse_expression(model_mock, cons['lhs'])
                    rhs = self._parse_expression(model_mock, cons['rhs'])
                    if cons['sense'] == '==':
                        if abs(lhs - rhs) > 1e-5: return False
                    elif cons['sense'] == '<=':
                        if lhs > rhs + 1e-5: return False
                    elif cons['sense'] == '>=':
                        if lhs < rhs - 1e-5: return False
                    else:
                        return False
                return True

            current = {}
            for var, props in vars_info.items():
                lb = props.get('lb', 0)
                ub = props.get('ub', 10)
                domain = props.get('domain', 'continuous')
                if domain == 'binary':
                    current[var] = random.choice([0, 1])
                elif domain == 'integer':
                    current[var] = random.randint(int(lb), int(ub))
                else:
                    current[var] = random.uniform(lb, ub)

            best = current.copy()
            best_val = evaluate(best) if is_feasible(best) else float('inf')

            T = 100.0
            T_min = 1e-3
            alpha = 0.95

            while T > T_min:
                for _ in range(100):
                    new = current.copy()
                    var_to_change = random.choice(list(vars_info.keys()))
                    domain = vars_info[var_to_change].get('domain', 'continuous')
                    lb = vars_info[var_to_change].get('lb', 0)
                    ub = vars_info[var_to_change].get('ub', 10)

                    if domain == 'binary':
                        new[var_to_change] = 1 - current[var_to_change]
                    elif domain == 'integer':
                        new[var_to_change] = max(min(current[var_to_change] + random.randint(-3, 3), ub), lb)
                    else:
                        new[var_to_change] = max(min(current[var_to_change] + random.uniform(-1, 1), ub), lb)

                    if not is_feasible(new):
                        continue

                    new_val = evaluate(new)
                    cur_val = evaluate(current)
                    delta = new_val - cur_val if sense == 'min' else cur_val - new_val

                    if delta < 0 or random.random() < math.exp(-delta / T):
                        current = new
                        if ((sense == 'min' and new_val < best_val) or
                            (sense == 'max' and new_val > best_val)):
                            best = new
                            best_val = new_val
                T *= alpha

            return {
                'status': 'success',
                'objective_value': best_val,
                'variables': {k: {'0': v} for k, v in best.items()},
                'solver_status': 'simulated_annealing',
                'termination_condition': 'final_temperature_reached'
            }

        except Exception as e:
            logger.error(f"Ошибка SA: {e}")
            return {'status': 'error', 'message': str(e)}

    
    def _parse_expression(self, model, expression):
        if isinstance(expression, (int, float)):
            return expression
            
        if isinstance(expression, str):
            var_name = expression
            if hasattr(model, var_name):
                return getattr(model, var_name)
            else:
                raise ValueError(f"Переменная '{var_name}' не найдена в модели")
                
        if isinstance(expression, dict):
            if 'op' not in expression:
                raise ValueError(f"Ключ 'op' отсутствует в выражении: {expression}")
                
            op = expression['op']
            
            if op == '+':
                return self._parse_expression(model, expression['left']) + self._parse_expression(model, expression['right'])
            elif op == '-':
                return self._parse_expression(model, expression['left']) - self._parse_expression(model, expression['right'])
            elif op == '*':
                return self._parse_expression(model, expression['left']) * self._parse_expression(model, expression['right'])
            elif op == '/':
                return self._parse_expression(model, expression['left']) / self._parse_expression(model, expression['right'])
            elif op == 'sum':
                if 'terms' not in expression:
                    raise ValueError(f"Ключ 'terms' отсутствует в операции sum: {expression}")
                return sum(self._parse_expression(model, term) for term in expression['terms'])
            else:
                raise ValueError(f"Неподдерживаемая операция: {op}")
                
        raise ValueError(f"Не удалось распознать выражение: {expression}")
        
    def _send_solution(self, task_id: str, solution: Dict[str, Any]):
        result = {
            'id': task_id,
            'solution': solution,
            'timestamp': str(uuid.uuid4())
        }
        
        try:
            self.producer.produce(
                self.output_topic,
                json.dumps(result).encode('utf-8'),
                callback=self._delivery_report
            )
            self.producer.flush()
            
            logger.info(f"Решение для задачи {task_id} отправлено в топик '{self.output_topic}'")
            
        except KafkaException as e:
            logger.error(f"Ошибка при отправке результата в Kafka: {e}")
            
    def _delivery_report(self, err, msg):
        if err is not None:
            logger.error(f"Ошибка доставки сообщения: {err}")
        else:
            logger.debug(f"Сообщение доставлено в {msg.topic()} [{msg.partition()}]")


def main():
    try:
        service = KafkaPyomoService(KAFKA_CONFIG)
        service.start()
    except Exception as e:
        logger.error(f"Критическая ошибка: {e}")
        return 1
    return 0

if __name__ == "__main__":
    exit(main())