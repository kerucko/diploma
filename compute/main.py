import json
from confluent_kafka import Consumer, Producer, KafkaException
from pyomo.environ import *
import logging


logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class PyomoKafkaSolver:
    def __init__(self, kafka_config):
        self.kafka_config = kafka_config
        self.consumer = None
        self.producer = None

    def setup_kafka(self):
        """Инициализация подключения к Kafka"""
        try:
            consumer_conf = {
                'bootstrap.servers': self.kafka_config['bootstrap_servers'],
                'group.id': self.kafka_config['group_id'],
                'auto.offset.reset': 'earliest'
            }
            self.consumer = Consumer(consumer_conf)
            
            producer_conf = {'bootstrap.servers': self.kafka_config['bootstrap_servers']}
            self.producer = Producer(producer_conf)
            
            self.consumer.subscribe([self.kafka_config['input_topic']])
            logger.info("Kafka подключен и готов к работе")
        except Exception as e:
            logger.error(f"Ошибка подключения к Kafka: {e}")
            raise

    @staticmethod
    def solve_optimization_problem(problem_data):
        """Решение оптимизационной задачи с Pyomo"""
        try:
            model = ConcreteModel()
            
            model.x = Var(within=NonNegativeReals)
            model.y = Var(within=NonNegativeReals)
            
            model.obj = Objective(expr=problem_data['coeff_x'] * model.x + 
                                  problem_data['coeff_y'] * model.y,
                                  sense=problem_data.get('sense', 'maximize'))
            
            model.constraint = Constraint(expr=model.x + model.y <= problem_data['constraint'])
            
            solver = SolverFactory('glpk')
            results = solver.solve(model)
            
            return {
                'status': str(results.solver.status),
                'termination_condition': str(results.solver.termination_condition),
                'objective_value': model.obj(),
                'variables': {
                    'x': model.x(),
                    'y': model.y()
                }
            }
        except Exception as e:
            logger.error(f"Ошибка решения задачи: {e}")
            raise

    def delivery_report(self, err, msg):
        """Callback для обработки статуса доставки сообщения"""
        if err is not None:
            logger.error(f"Ошибка доставки сообщения: {err}")
        else:
            logger.info(f"Сообщение доставлено в {msg.topic()} [{msg.partition()}]")

    def process_message(self, msg):
        """Обработка входящего сообщения"""
        try:
            data = json.loads(msg.value().decode('utf-8'))
            task_id = data['id']
            problem = data['problem']
            
            logger.info(f"Получена задача ID: {task_id}")
            
            solution = self.solve_optimization_problem(problem)
            
            response = {
                'id': task_id,
                'solution': solution
            }
            
            self.producer.produce(
                self.kafka_config['output_topic'],
                json.dumps(response).encode('utf-8'),
                callback=self.delivery_report
            )
            self.producer.poll(0)
            
            logger.info(f"Задача ID: {task_id} успешно обработана")
            
        except json.JSONDecodeError:
            logger.error("Ошибка декодирования JSON")
        except KeyError as e:
            logger.error(f"Отсутствует обязательное поле: {e}")
        except Exception as e:
            logger.error(f"Ошибка обработки сообщения: {e}")

    def run(self):
        """Основной цикл обработки сообщений"""
        try:
            self.setup_kafka()
            
            logger.info("Сервис запущен. Ожидание сообщений...")
            while True:
                msg = self.consumer.poll(1.0)
                
                if msg is None:
                    continue
                if msg.error():
                    raise KafkaException(msg.error())
                
                self.process_message(msg)
                
        except KeyboardInterrupt:
            logger.info("Остановка сервиса...")
        finally:
            self.consumer.close()
            self.producer.flush()
            logger.info("Ресурсы освобождены")

if __name__ == "__main__":
    config = {
        'bootstrap_servers': 'localhost:9092',
        'group_id': 'pyomo-solver-group',
        'input_topic': 'tasks',
        'output_topic': 'completed'
    }
    
    solver_service = PyomoKafkaSolver(config)
    solver_service.run()